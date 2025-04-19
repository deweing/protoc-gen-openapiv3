WORKDIR_IN_DOCKER = /go/src/github.com/deweing/protoc-gen-openapiv3
INCLUDE_IN_DOCKER = -I=/go/src
GOPATH := $(shell go env GOPATH | awk -F ":" '{print $$1}')
GOROOT := $(shell go env GOROOT)
LOCAL_SRC_PATH = ${PWD}/../../../
INCLUDE = -I=${GOPATH}/src -I=${LOCAL_SRC_PATH} -I=${GOPATH}/src/github.com/googleapis/googleapis

proto:
	@protoc $(INCLUDE) --go_opt=paths=source_relative --proto_path=./swagger --go_out=./swagger ./swagger/openapiv3.proto
	@protoc $(INCLUDE) --go_opt=paths=source_relative --proto_path=./swagger --go_out=./swagger ./swagger/annotations.proto
	@protoc $(INCLUDE) --go_opt=paths=source_relative --proto_path=./internal/descriptor/openapiconfig/ --go_out=./internal/descriptor/openapiconfig ./internal/descriptor/openapiconfig/openapiconfig.proto
	sed -i "" 's/file_github_com_deweing_protoc_gen_openapiv3_swagger_openapiv3_proto_init/file_openapiv3_proto_init/g' ./swagger/annotations.pb.go

build:
	go build -ldflags '-w -s' -o protoc-gen-openapiv3 .

install:
	go install .

example:
	@protoc $(INCLUDE) \
			--proto_path=./testdata/store \
    		--openapiv3_out . \
    		--openapiv3_opt logtostderr=true \
    		--openapiv3_opt json_names_for_fields=true \
    		--openapiv3_opt disable_default_errors=true \
    		--openapiv3_opt allow_merge=true \
    		--openapiv3_opt output_format=json \
    		./testdata/store/*.proto

clean:
	rm -f apidocs.swagger.yaml apidocs.swagger.json

test: clean install example

.PHONY: proto install build example test clean