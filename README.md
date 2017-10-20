
# ec2-exporter

First Step :
```
cp docker-compose.yml.sample docker-compose.yml
```

add your AWS credentials in docker-compose.yml :

and start :
```
docker-compose up
```

you can now acces to metrics :
```
http://0.0.0.0:9599/metrics
```
