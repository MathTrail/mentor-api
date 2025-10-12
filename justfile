build: docker build -t mathtrail-mentor .

deploy:
    helm upgrade --install mathtrail-mentor ./helm/mathtrail-mentor --values ./helm/mathtrail-mentor/values.yaml
    kubectl rollout status deployment/mathtrail-mentor
    kubectl get svc
    kubectl get pods

test: minikube service mathtrail-mentor --url | xargs -I{} curl {}/hello
