module paidpiper.com/payment-gateway

go 1.12

require (
	github.com/deepmap/oapi-codegen v1.6.1
	github.com/go-chi/chi v4.0.3+incompatible
	github.com/go-chi/chi/v5 v5.0.2
	github.com/go-errors/errors v0.0.0-20150906023321-a41850380601
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.3
	github.com/google/go-cmp v0.4.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/handlers v1.5.0
	github.com/gorilla/mux v1.8.0
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.7
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/rs/xid v1.2.1
	github.com/stellar/go v0.0.0-20210324164845-827227e3edd3
	github.com/stretchr/testify v1.5.1
	github.com/tkanos/gonfig v0.0.0-20181112185242-896f3d81fadf
	go.opentelemetry.io/otel v0.4.2
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.4.2
	google.golang.org/grpc v1.27.1
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	paidpiper.com/provider-service v0.0.0
)

replace paidpiper.com/provider-service => ./../payment-gateway-webui
