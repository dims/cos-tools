COS_CONVERTER_BINARY=cos_ova_converter

build:
	go build -o ${COS_CONVERTER_BINARY} .

clean:
	go clean
	rm -f ${COS_CONVERTER_BINARY}

build-image:
	cd ../../../ && docker build -t ${COS_CONVERTER_BINARY} -f src/cmd/cos_ova_converter/Dockerfile .
