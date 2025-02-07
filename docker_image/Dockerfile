FROM dockurr/windows:latest

COPY ./scripts/* /oem/ 

ENV USERNAME=daytona
ENV PASSWORD=daytona

EXPOSE 8006 3389 22 2222


ENTRYPOINT ["/usr/bin/tini" , "-s", "/run/entry.sh" ]