package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "regexp"
    "time"

    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/zipkin"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func initTracer() {
    exporter, err := zipkin.New(
        "http://zipkin:9411/api/v2/spans",
    )
    if err != nil {
        log.Fatalf("failed to initialize zipkin exporter %v", err)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.Default()),
    )
    otel.SetTracerProvider(tp)
    tracer = tp.Tracer("service-B")
}

type CEPRequest struct {
    CEP string `json:"cep"`
}

type CEPResponse struct {
    Localidade string `json:"localidade"`
}

type WeatherResponse struct {
    Current struct {
        TempC float64 `json:"temp_c"`
    } `json:"current"`
}

type FinalResponse struct {
    City   string  `json:"city"`
    TempC  float64 `json:"temp_C"`
    TempF  float64 `json:"temp_F"`
    TempK  float64 `json:"temp_K"`
}

func validateCEP(cep string) bool {
    re := regexp.MustCompile(`^\d{8}$`)
    return re.MatchString(cep)
}

func handler(w http.ResponseWriter, r *http.Request) {
    ctx, span := tracer.Start(r.Context(), "Service B - Handler")
    defer span.End()

    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    var cepRequest CEPRequest
    err = json.Unmarshal(body, &cepRequest)
    if err != nil || !validateCEP(cepRequest.CEP) {
        http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
        return
    }

    // Get City Name
    city, err := getCity(ctx, cepRequest.CEP)
    if err != nil {
        http.Error(w, "can not find zipcode", http.StatusNotFound)
        return
    }

    // Get Weather
    tempC, err := getWeather(ctx, city)
    if err != nil {
        http.Error(w, "can not get weather", http.StatusInternalServerError)
        return
    }

    tempF := tempC*1.8 + 32
    tempK := tempC + 273.15

    response := FinalResponse{
        City:  city,
        TempC: tempC,
        TempF: tempF,
        TempK: tempK,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func getCity(ctx context.Context, cep string) (string, error) {
    _, span := tracer.Start(ctx, "Get City")
    defer span.End()

    resp, err := http.Get(fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return "", fmt.Errorf("CEP not found")
    }
    var cepResp CEPResponse
    body, _ := ioutil.ReadAll(resp.Body)
    json.Unmarshal(body, &cepResp)
    if cepResp.Localidade == "" {
        return "", fmt.Errorf("CEP not found")
    }
    return cepResp.Localidade, nil
}

func getWeather(ctx context.Context, city string) (float64, error) {
    _, span := tracer.Start(ctx, "Get Weather")
    defer span.End()

    apiKey := os.Getenv("WEATHER_API_KEY")
    url := fmt.Sprintf("http://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no", apiKey, city)
    resp, err := http.Get(url)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return 0, fmt.Errorf("Weather not found")
    }
    var weatherResp WeatherResponse
    body, _ := ioutil.ReadAll(resp.Body)
    json.Unmarshal(body, &weatherResp)
    return weatherResp.Current.TempC, nil
}

func main() {
    initTracer()
    mux := http.NewServeMux()
    mux.HandleFunc("/cep", handler)
    handler := otelhttp.NewHandler(mux, "Service B")
    log.Println("Service B is running on port 8081")
    log.Fatal(http.ListenAndServe(":8081", handler))
}
