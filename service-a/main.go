package main

import (
    "context"
    "encoding/json"
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
    tracer = tp.Tracer("service-A")
}

type CEPRequest struct {
    CEP string `json:"cep"`
}

func validateCEP(cep string) bool {
    re := regexp.MustCompile(`^\d{8}$`)
    return re.MatchString(cep)
}

func handler(w http.ResponseWriter, r *http.Request) {
    ctx, span := tracer.Start(r.Context(), "Service A - Handler")
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

    // Forward to Service B
    client := http.Client{
        Transport: otelhttp.NewTransport(http.DefaultTransport),
    }
    reqBody, _ := json.Marshal(cepRequest)
    req, err := http.NewRequestWithContext(ctx, "POST", "http://service-b:8081/cep", ioutil.NopCloser(r.Body))
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }
    req.Header.Set("Content-Type", "application/json")
    req.Body = ioutil.NopCloser(r.Body)
    resp, err := client.Do(req)
    if err != nil {
        http.Error(w, "service unavailable", http.StatusServiceUnavailable)
        return
    }
    defer resp.Body.Close()
    w.WriteHeader(resp.StatusCode)
    responseBody, _ := ioutil.ReadAll(resp.Body)
    w.Write(responseBody)
}

func main() {
    initTracer()
    mux := http.NewServeMux()
    mux.HandleFunc("/cep", handler)
    handler := otelhttp.NewHandler(mux, "Service A")
    log.Println("Service A is running on port 8080")
    log.Fatal(http.ListenAndServe(":8080", handler))
}
