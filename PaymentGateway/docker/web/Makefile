build:
	docker build \
		-t web_hoster_stage \
		-t torplusserviceregistry.azurecr.io/private/web_hoster_stage:latest \
		-f ./Dockerfile  ../../../
build_prod:
	docker build \
		-t web_hoster_prod \
		-t torplusserviceregistry.azurecr.io/private/web_hoster_prod:latest \
		-f ./Dockerfile  ../../../
run_it:
	docker run --rm -p 80:80 \
					-e seed=SAGWKBJQ5J2AJAM5YTEUGSOEXUBOVNZJ6FNJDN7CBKZ543VRQ7XTDFIQ \
					-v $(PWD)/www:/opt/www \
					-v $(PWD)/tor:/root/tor \
					-it --entrypoint /bin/bash  \
	 				torplusserviceregistry.azurecr.io/private/web_hoster_stage
run_prod_it:
	docker run --rm -p 80:80 \
					-e seed=SAGWKBJQ5J2AJAM5YTEUGSOEXUBOVNZJ6FNJDN7CBKZ543VRQ7XTDFIQ \
					-v $(PWD)/www:/opt/www \
					-v $(PWD)/tor:/root/tor \
					-it --entrypoint /bin/bash \
					torplusserviceregistry.azurecr.io/private/web_hoster_prod
run:
	docker run --rm \
		-e seed=SAGWKBJQ5J2AJAM5YTEUGSOEXUBOVNZJ6FNJDN7CBKZ543VRQ7XTDFIQ \
		-p 80:80 \
		-v $(PWD)/www:/opt/www \
		-v $(PWD)/tor:/root/tor \
		torplusserviceregistry.azurecr.io/private/web_hoster_stage:latest
run_prod:
	docker run --rm \
		-e seed=SAGWKBJQ5J2AJAM5YTEUGSOEXUBOVNZJ6FNJDN7CBKZ543VRQ7XTDFIQ \
		-p 80:80 \
		-v $(PWD)/www:/opt/www \
		-v $(PWD)/tor:/root/tor \
		torplusserviceregistry.azurecr.io/private/web_hoster_prod:latest
push:
	docker push torplusserviceregistry.azurecr.io/private/web_hoster_stage:latest
push_prod:
	docker push torplusserviceregistry.azurecr.io/private/web_hoster_prod:latest
