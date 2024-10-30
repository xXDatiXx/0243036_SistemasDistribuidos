# Esto nos ayuda a posicionar nuestros config files en una carpeta dentro de nuestro proyecto

CONFIG_PATH=/home/dati/Desktop/UP/0243036_SistemasDistribuidos

.PHONY: init

init:
	mkdir -p ${CONFIG_PATH}

.PHONY: gencert
# gencert
# First creates the bare certificate, it is just the base certificate that others will differ from
# Then creates the server certificate, this allows our server certification
# Finally we create the client certificate this allows two way authentication
gencert:
	cfssl gencert \
		-initca test/ca-csr.json | cfssljson -bare ca

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=server \
		test/server-csr.json | cfssljson -bare server

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		test/client-csr.json | cfssljson -bare client

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		-cn="root" \
		test/client-csr.json | cfssljson -bare root-client

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		-cn="nobody" \
		test/client-csr.json | cfssljson -bare nobody-client
	mv *.pem *.csr ${CONFIG_PATH}

compile:
	protoc api/v1/*.proto \
					--go_out=.\
					--go_opt=paths=source_relative \
					--proto_path=.
$(CONFIG_PATH)/model.conf:
	cp test/model.conf $(CONFIG_PATH)/model.conf

$(CONFIG_PATH)/policy.csv:
	cp test/policy.csv $(CONFIG_PATH)/policy.csv

test: $(CONFIG_PATH)/policy.csv $(CONFIG_PATH)/model.conf
	go test -race ./...
compile_rpc:
	protoc api/v1/*.proto \
	--go_out=. \
	--go_opt=paths=source_relative \
    --go-grpc_out=. \
	--go-grpc_opt=paths=source_relative \
	--proto_path=.