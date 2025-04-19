# Protoc-Gen-Openapi

## About

**注意**: 该项目目前完成50%左右，不能完全支持openapi3.0的所有格式，需根据自己的情况决定是否使用；

基于 [grpc-ecosystem/grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) 修改；
由于 [grpc-ecosystem/grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) 当前代码生成的swagger还是2.0的版本；
所以在此基础上修改代码，生成openapi3.x;

## 依赖
安装运行时需要依赖以下文件：可以从 [googleapis 仓库](https://github.com/googleapis/googleapis) 下载
```shell
google/api/annotations.proto
google/api/field_behavior.proto
google/api/http.proto
google/api/httpbody.proto
```

[go.mod](go.mod)
```shell
github.com/golang/protobuf v1.5.2
```

## Install
```shell
make install
```

## 插件开发

**注意**：如需开发，protoc版本为1.27.1及以上;注意与当前开发版本冲突问题；推荐使用docker编译

## Example

Demo [store](testdata/store)

**注意**：执行命令时，注意proto文件的引入路径；参考以下列子，`LOCAL_SRC_PATH`根据自己的环境设置
```shell
GOPATH := $(shell go env GOPATH | awk -F ":" '{print $$1}')
LOCAL_SRC_PATH = /workspace_path
INCLUDE = -I=${GOPATH} -I=${GOPATH}/src -I=${LOCAL_SRC_PATH} -I=${GOPATH}/src/github.com/googleapis/googleapis
```

```shell
make example
```

## 使用说明

注意：
- 在需要定义API文档的proto的文件里面引入`github.com/deweing/protoc-gen-openapiv3/swagger/annotations.proto`文件；
- 需要定义go_package: `option go_package="github.com/deweing/store-srv/proto/store-srv";`; 在编译成go文件时如果不需要创建路径，则需要带上`--go_opt=paths=source_relative`参数；
- go.mod中需要替换以下包，删除`google.golang.org/grpc/examples`包

```shell
replace (
	google.golang.org/grpc v1.46.0 => google.golang.org/grpc v1.24.0
)
```

### swagger info
在任意proto文件头部声明，建议统一放到common.proto里面
```shell
option (swagger.swagger) = {
  info: {
    title: "Doc.example-srv";
    version: "0.0.1";
    contact: {
      name: "Tom";
      email: "tom@example.com";
    };
  };

  servers: [
    {
      url: "https://gateway.example.com/api/example-srv",
      description: "测试"
    },
    {
      url: "https://ygateway.exmaple.com/api/example-srv",
      description: "预发"
    },
    {
      url: "https://gateway.exmaple.com/api/example-srv",
      description: "线上"
    }
  ]

  external_docs: {
    url: "http://docs.exmaple.com/project/128",
    description: "External Docs"
  }

  responses: {
    key: "400";
    value: {
      description: "A failed response"
      content: {
        key: "application/json",
        value: {
          schema: {
            json_schema: {
              ref: ".example_srv.ErrorRsp"
            }
          }
        }
      }
    }
  }
};
```

如果统一定义错误的返回信息，需要在pb中定义一个错误的message
```shell
message ErrorRsp {
  int64 code = 1 [(gogoproto.jsontag) = "code", (gogoproto.nullable) = true, (swagger.field) = {description: "返回代码: 200正常,其他错误", example: "100100100404"}];
  string detail = 2 [(gogoproto.jsontag) = "detail", (gogoproto.nullable) = true, (swagger.field) = {description: "错误信息", example: "\"page not found\""}];
}
```

### swagger path

#### 路径
在`service`里面定义，需要引入`google/api/annotations.proto`;

post请求参数不能同时声明到body和query里面；
- `body`为空，所有未被路径捕获的请求字段都会声明到query上
- `body`为`*`，所有未被路径捕获的请求字段都会声明到body里面

```shell
service Store {
  option (swagger.tag) = {description: "门店", sort:1};

  // 创建门店
  rpc Create(StoreCreateReq) returns (StoreCreateRsp) {
    option (google.api.http) = {
      post: "/v1/Store/Create",
      body: "*"
    };
  }

  // 获取门店
  rpc Get(StoreIdReq) returns (StoreGetRsp) {
    option (google.api.http) = {
      get: "/v1/Store/Get"
    };
  }

  // 删除门店
  rpc Delete(StoreIdReq) returns (ResultRsp) {
    option (google.api.http) = {
      delete: "/v1/Store/Delete",
    };

    option (swagger.operation)={deprecated: true};
  }

  // 更新门店
  rpc Update(StoreUpdateReq) returns (ResultRsp) {
    option (google.api.http) = {
      post: "/v1/Store/Update/{storeId}",
      body: "*"
    };
  }
}
```

#### 请求参数：

字段的Type和Format会通过proto的字段类型映射出来，如果没有特殊需要，可以不用设置；
如果设置为指定的类型可以通过`type`指定，数组的值类型通过`item_type`设置；

建议所有的字段都设置`description`和`example`标签，以便生成的文档易于理解；

字段`requried`属性可以统一在message的option上设置；

注意：**example**的类型为`json.RawMessage`,如果需要设置为字符串时，需要加`"`；如：`example: "\"新新超市\""`;
如果为数组`example: "\"[\\\"手机\\\",\\\"电脑\\\"]\"`；Json字符串：`example: "\"{\\\"relatedID\\\":123456,\\\"value\\\":99.99}\""`

```shell
message StoreCreateReq {
  option (swagger.schema) = {
    json_schema: {
      description: "创建门店",
      required: ["appId", "accountId", "storeName"]
    }
  };

  uint32 appId = 1 [(swagger.field) = {description: "产品ID", example: "1"}];
  uint64 accountId = 2 [(swagger.field) = {description: "主账号ID", type: STRING, example: "100010"}];
  string storeName = 3 [(swagger.field) = {description: "门店名称", example: "\"青羊全家超市\""}];
  StoreType storeType = 4;
  uint32 areaCode = 5 [(swagger.field) = {description: "地区代码", example:"501010"}];
  string address = 6 [(swagger.field) = {description: "详细地址", example: "\"春熙路银石广场108楼\""}];
  string storePhone = 7 [(swagger.field) = {description: "门店联系电话", example: "\"18800008888\""}];
  string contactName = 8 [(swagger.field) = {description: "联系人", example: "\"李元芳\""}];
  string contactPhone = 9 [(swagger.field) = {description: "联系人电话", example: "\"18800008888\""}];
  string comment = 10 [(swagger.field) = {description: "备注", example: "\"备注\""}];
  StoreStatus status = 11;
  StoreSource source = 12;
  int64 sourceId = 13 [(swagger.field) = {description: "来源账号ID", example: "123456"}];
}
```

#### 响应参数：
同请求参数基本一致
```shell
message StoreListRsp {
  int64 code = 1 [(gogoproto.jsontag) = "code", (gogoproto.nullable) = true, (swagger.field) = {description: "返回代码:200正常,其他错误", example: "200"}];
  string detail = 2 [(gogoproto.jsontag) = "detail", (gogoproto.nullable) = true, (swagger.field) = {description: "错误信息"}];
  // 返回数据
  Data data = 3 [(gogoproto.jsontag) = "data", (gogoproto.nullable) = true];

  message Data {
    // 门店
    repeated StoreEntity stores = 1 [(gogoproto.jsontag) = "stores", (gogoproto.nullable) = true,  (swagger.field) = {description: "门店"}];
  }
}应
```
### 生成文档
替换`INCLUDE`和`PROJECT`为自己路径
```sehell
protoc $(INCLUDE) \
    --proto_path=./proto/${PROJECT} \
    --openapiv3_out . \
    --openapiv3_opt logtostderr=true \
    --openapiv3_opt json_names_for_fields=true \
    --openapiv3_opt disable_default_errors=true \
    --openapiv3_opt allow_merge=true \
    --openapiv3_opt merge_file_name=openapi \
    --openapiv3_opt enums_as_ints=true \
    --openapiv3_opt output_format=json \
    ./proto/${PROJECT}/*.proto
```

部分参数说明:
- `allow_merge`：合并生成的文件
- `merge_file_name`: 合并文件前缀
- `output_format`: 生成的文件格式: 支持`json`和`yaml`
- `enums_as_ints`: true将enum转换成int,否则为false；默认false

## 参考
- [grpc-ecosystem/grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway)
- [OpenAPI](https://swagger.io/docs/specification/about/)
- [googleapis 仓库](https://github.com/googleapis/googleapis)