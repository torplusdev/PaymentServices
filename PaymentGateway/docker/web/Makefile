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
	docker run --rm -e PP_ENV=stage -e role=hs_client -p 80:80 -v $(PWD)/www:/opt/www -it --entrypoint /bin/bash tor_site_hoster
run:
	docker run --rm \
		-e PP_ENV=stage \
		-e role=hs_client \
		-e seed=SAGWKBJQ5J2AJAM5YTEUGSOEXUBOVNZJ6FNJDN7CBKZ543VRQ7XTDFIQ
		-p 80:80 -v $(PWD)/www:/opt/www tor_site_hoster
push:
	docker push torplusserviceregistry.azurecr.io/private/web_hoster_stage:latest
push_prod:
	docker push torplusserviceregistry.azurecr.io/private/web_hoster_prod:latest
