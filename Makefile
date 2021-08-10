build:
	docker build -t limaoim4 .
push:
	docker tag limaoim4 registry.cn-shanghai.aliyuncs.com/limao/limaoim4:latest
	docker push registry.cn-shanghai.aliyuncs.com/limao/limaoim4
deploy:
	docker build -t limaoim4 .
	docker tag limaoim4 registry.cn-shanghai.aliyuncs.com/limao/limaoim4:latest
	docker push registry.cn-shanghai.aliyuncs.com/limao/limaoim4