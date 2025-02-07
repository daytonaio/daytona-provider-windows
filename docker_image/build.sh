 docker kill $(docker ps -a --filter ancestor=rutik7066/daytona-windows-container -q ) && \
docker rm $(docker ps -a --filter ancestor=rutik7066/daytona-windows-container -q ) && \
docker rmi $(docker images rutik7066/daytona-windows-container -q) -f && \
docker rmi rutik7066/daytona-windows-container:latest -f && \
docker build -t rutik7066/daytona-windows-container:latest -f Dockerfile . && \ 
docker push rutik7066/daytona-windows-container:latest