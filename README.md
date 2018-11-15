# duplayer
A tool for discovering how to reduce the size of Docker Image.
It shows how much image size is reduced when each layer is merged and which files are duplicated.

## Installation
```bash
go get github.com/recruit-tech/duplayer
```

## How to use
```bash 
docker save image:tag | duplayer | less
```
or
```bash
docker save -o image.tar image:tag
duplayer -f image.tar | less
```

## License
MIT