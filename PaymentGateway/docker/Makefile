VERSION:=$(shell git describe --tags --dirty --always)
build_image:
	docker build -f ./Build.Dockerfile \
				-t pg_build ../../ 
image: 
	docker build --build-arg PG_VERSION=$(VERSION) -f ./Dockerfile -t paymentgateway \
	-t torplusserviceregistry.azurecr.io/private/paymentgateway:$(VERSION) \
	-t torplusserviceregistry.azurecr.io/private/paymentgateway:latest \
	../../
push: image
	docker push torplusserviceregistry.azurecr.io/private/paymentgateway:$(VERSION) 
	docker push torplusserviceregistry.azurecr.io/private/paymentgateway:latest 
run: 
	docker run -p 28080:28080 --rm torplusserviceregistry.azurecr.io/private/paymentgateway:$(VERSION)
run_it:
	docker run -it -p 8080:8080 -p 28080:28080 --name pg --entrypoint /bin/bash --rm torplusserviceregistry.azurecr.io/private/paymentgateway:$(VERSION)
attach:
	docker exec -it pg bash
