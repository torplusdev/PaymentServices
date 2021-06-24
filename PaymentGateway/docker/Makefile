VERSION:=$(shell git describe --tags --dirty --always)
build_image:
	docker build -f ./Build.Dockerfile \
				-t pg_build ../../ 
image: 
	docker build --build-arg PG_VERSION=$(VERSION) -f ./Dockerfile -t paymentgateway \
	-t torplusserviceregistry.azurecr.io/private/paymentgateway:$(VERSION) \
	-t torplusserviceregistry.azurecr.io/private/paymentgateway:latest \
	../../
az_pipeline_image:
	docker build --build-arg PG_VERSION=$(VERSION) -f ./Dockerfile_az_pipeline --no-cache \
	-t paymentgateway \
	-t torplusserviceregistry.azurecr.io/private/paymentgateway:$(VERSION) \
	-t torplusserviceregistry.azurecr.io/private/paymentgateway:latest \
	../../
push: image
	docker push torplusserviceregistry.azurecr.io/private/paymentgateway:$(VERSION) 
	docker push torplusserviceregistry.azurecr.io/private/paymentgateway:latest 
az_pipeline_push: az_pipeline_image
	docker push torplusserviceregistry.azurecr.io/private/paymentgateway:$(VERSION)
	docker push torplusserviceregistry.azurecr.io/private/paymentgateway:latest
clean_up:
	docker rmi --force paymentgateway \
	torplusserviceregistry.azurecr.io/private/paymentgateway:$(VERSION) \
	torplusserviceregistry.azurecr.io/private/paymentgateway:latest \
	torplusserviceregistry.azurecr.io/private/tor_plus:latest
	docker image prune --force
run: 
	docker run -p 28080:28080 --rm torplusserviceregistry.azurecr.io/private/paymentgateway:$(VERSION)
run_it:
	docker run -it -p 8080:8080 -p 28080:28080 --name pg --entrypoint /bin/bash --rm torplusserviceregistry.azurecr.io/private/paymentgateway:$(VERSION)
attach:
	docker exec -it pg bash
