fwd-registry:
	docker run --rm -it --network=host alpine ash -c "apk add socat && socat TCP-LISTEN:5000,reuseaddr,fork TCP:$(minikube ip):5000"

DOCKER_REPO=localhost:5000

build:
	KO_DOCKER_REPO=$(DOCKER_REPO) ko build ./cmd/sloth/ --base-import-paths

strategies = ah jumbo
apply:
	kubectl apply -f .kubernetes/ns.yml
	kubectl apply -f .kubernetes/services/db/deployment.yml
	kubectl apply -f .kubernetes/services/pubsub/deployment.yml
	for strat in $(strategies); do \
		kustomize build .kubernetes/overlays/$$strat | kubectl apply -f - ; \
	done;
