FROM golang:1.19-alpine
# COPY collector /home/collector
COPY . /home/collector
WORKDIR /home/collector
# RUN chmod +x collector
CMD [ "go", "run", "." ]