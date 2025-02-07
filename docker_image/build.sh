 docker kill $(docker ps -a --filter ancestor=daytonaio/workspace-windows -q ) && \
docker rm $(docker ps -a --filter ancestor=daytonaio/workspace-windows -q ) && \
docker rmi $(docker images daytonaio/workspace-windows -q) -f && \
docker rmi daytonaio/workspace-windows:latest -f && \
docker build -t daytonaio/workspace-windows:latest -f Dockerfile . && \ 
docker push daytonaio/workspace-windows:latest