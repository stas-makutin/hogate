set GOOS=linux
set GOARCH=arm64
go build -o bootstrap main.go
%GOPATH%\bin\build-lambda-zip.exe -output hogatefn.zip bootstrap