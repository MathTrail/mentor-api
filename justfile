build: docker build -t mathtrail-mentor .

deploy:
    helm upgrade --install mathtrail-mentor ./helm/mathtrail-mentor --values ./helm/mathtrail-mentor/values.yaml
    kubectl rollout status deployment/mathtrail-mentor
    kubectl get svc
    kubectl get pods

test:
    kubectl port-forward svc/mathtrail-mentor 8080:80 &
    sleep 2
    curl http://localhost:8080/hello
