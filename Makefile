
.PHONY: gencert
gencert:
#	cfssl gencert \
#		-initca test/ca-csr.json | cfssljson -bare ca
#
#	cfssl gencert \
#		  -ca=ca.pem \
#		  -ca-key=ca-key.pem \
#		  -config=test/ca-config.json \
#  		  -profile=server \
#  		  test/server-csr.json | cfssljson -bare server
#
#	cfssl gencert \
#		-ca=ca.pem \
#		-ca-key=ca-key.pem \
#		-config=test/ca-config.json \
#		-profile=client \
#		test/client-csr.json | cfssljson -bare client

	cfssl gencert \
			-ca=test/ca.pem \
			-ca-key=test/ca-key.pem \
			-config=test/ca-config.json \
			-profile=client \
			-cn="root" \
			test/client-csr.json | cfssljson -bare root-client

	cfssl gencert \
			-ca=test/ca.pem \
			-ca-key=test/ca-key.pem \
			-config=test/ca-config.json \
			-profile=client \
			-cn="nobody" \
			test/client-csr.json | cfssljson -bare nobody-client
