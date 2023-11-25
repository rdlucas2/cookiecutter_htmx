### work locally
```
docker build -t gohtmx:local --target local .
docker run -it --rm -v "$(pwd):/working" -p 3000:3000 --name gohtmx gohtmx:local
```

### test
```
docker build -t gohtmx:test --target test .
docker run -it --rm -v "$(pwd)/out:/out" --name gohtmx gohtmx:test
```

### sonar scan
```
docker run -it --rm -e SONAR_HOST_URL="http://host.docker.internal:9000" -e SONAR_LOGIN="<your-generated-token>" -v "$(pwd):/usr/src" sonarsource/sonar-scanner-cli
```

### run
```
docker build -t gohtmx:run --target run .
docker run -it --rm --name gohtmx gohtmx:run
```

### artifact
```
docker build -t gohtmx:latest --target artifact .
docker run -it --rm -p 3000:3000 --name gohtmx gohtmx:latest
```

### trivy scan
```
docker run -it --rm -v /var/run/docker.sock:/var/run/docker.sock -v "$(pwd)/out:/out" aquasec/trivy image --format table --output /out/trivy-report.txt --scanners vuln gohtmx:latest
```
