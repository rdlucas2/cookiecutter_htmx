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

### run
```
docker build -t gohtmx:latest --target run .
docker run -it --rm --name gohtmx gohtmx:latest
```

### artifact
```
docker build -t gohtmx:artifact --target artifact .
docker run -it --rm -v "$(pwd)/artifact:/artifact --name gohtmx gohtmx:artifact
```

