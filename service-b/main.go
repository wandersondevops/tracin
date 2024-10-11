package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "net/url"
    "os"
    "regexp"

    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/zipkin"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
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
        sdktrace.WithResource(resource.NewSchemaless(
            semconv.ServiceNameKey.String("service-B"),
        )),
    )
    otel.SetTracerProvider(tp)
    tracer = otel.Tracer("service-B")
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
    City  string  `json:"city"`
    TempC float64 `json:"tempC"`
    TempF float64 `json:"tempF"`
    TempK float64 `json:"tempK"`
}

func validateCEP(cep string) bool {
    re := regexp.MustCompile(`^\d{8}$`)
    return re.MatchString(cep)
}

func handler(w http.ResponseWriter, r *http.Request) {
    ctx, span := tracer.Start(r.Context(), "Service B - Handler")
    defer span.End()

    body, err := io.ReadAll(r.Body)
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
        log.Println("Error getting city:", err)
        http.Error(w, "cannot find zipcode", http.StatusNotFound)
        return
    }

    // Get Weather
    tempC, err := getWeather(ctx, city)
    if err != nil {
        log.Println("Error getting weather:", err)
        http.Error(w, fmt.Sprintf("cannot get weather: %v", err), http.StatusInternalServerError)
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
    ctx, span := tracer.Start(ctx, "Get City")
    defer span.End()

    client := http.Client{
        Transport: otelhttp.NewTransport(http.DefaultTransport),
    }
    url := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep)
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        log.Println("Error creating request to ViaCEP:", err)
        return "", err
    }
    resp, err := client.Do(req)
    if err != nil {
        log.Println("Error making request to ViaCEP:", err)
        return "", err
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        log.Printf("ViaCEP returned status %d: %s\n", resp.StatusCode, string(body))
        return "", fmt.Errorf("ViaCEP returned status %d", resp.StatusCode)
    }
    var cepResp CEPResponse
    body, _ := io.ReadAll(resp.Body)
    err = json.Unmarshal(body, &cepResp)
    if err != nil {
        log.Println("Error parsing JSON from ViaCEP:", err)
        return "", err
    }
    if cepResp.Localidade == "" {
        log.Println("Localidade not found in ViaCEP response")
        return "", fmt.Errorf("CEP not found in ViaCEP")
    }
    return cepResp.Localidade, nil
}

func getWeather(ctx context.Context, city string) (float64, error) {
    ctx, span := tracer.Start(ctx, "Get Weather")
    defer span.End()

    apiKey := os.Getenv("WEATHER_API_KEY")
    if apiKey == "" {
        log.Println("WEATHER_API_KEY is not set")
        return 0, fmt.Errorf("WEATHER_API_KEY is not set")
    }

    // URL-encode the city name
    encodedCity := url.QueryEscape(city)
    url := fmt.Sprintf("http://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no", apiKey, encodedCity)
    log.Println("Requesting WeatherAPI with URL:", url)

    client := http.Client{
        Transport: otelhttp.NewTransport(http.DefaultTransport),
    }
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        log.Println("Error creating request to WeatherAPI:", err)
        return 0, err
    }
    resp, err := client.Do(req)
    if err != nil {
        log.Println("Error making request to WeatherAPI:", err)
        return 0, err
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        log.Printf("WeatherAPI returned status %d: %s\n", resp.StatusCode, string(body))
        return 0, fmt.Errorf("WeatherAPI error: %s", string(body))
    }
    log.Println("WeatherAPI response:", string(body))
    var weatherResp WeatherResponse
    err = json.Unmarshal(body, &weatherResp)
    if err != nil {
        log.Println("Error parsing JSON from WeatherAPI:", err)
        return 0, err
    }
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
