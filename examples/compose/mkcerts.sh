#!/bin/sh

set -eu
export LC_ALL='C'

CERTS_DIR="$(CDPATH='' cd -- "$(dirname -- "${0:?}")" && pwd -P)"/certs/

mkdir -p "${CERTS_DIR:?}"/ca/
CA_KEY="${CERTS_DIR:?}"/ca/key.pem
CA_CRT="${CERTS_DIR:?}"/ca/cert.pem

mkdir -p "${CERTS_DIR:?}"/server/
SERVER_KEY="${CERTS_DIR:?}"/server/key.pem
SERVER_CSR="${CERTS_DIR:?}"/server/csr.pem
SERVER_CRT="${CERTS_DIR:?}"/server/cert.pem
SERVER_CA_CRT="${CERTS_DIR:?}"/server/ca.pem
SERVER_OPENSSL_CNF="${CERTS_DIR:?}"/server/openssl.cnf

mkdir -p "${CERTS_DIR:?}"/client/
CLIENT_KEY="${CERTS_DIR:?}"/client/key.pem
CLIENT_CSR="${CERTS_DIR:?}"/client/csr.pem
CLIENT_CRT="${CERTS_DIR:?}"/client/cert.pem
CLIENT_CA_CRT="${CERTS_DIR:?}"/client/ca.pem
CLIENT_OPENSSL_CNF="${CERTS_DIR:?}"/client/openssl.cnf

if [ ! -e "${CA_KEY:?}" ] || ! openssl rsa -check -in "${CA_KEY:?}" -noout >/dev/null 2>&1; then
	printf '%s\n' 'Generating CA private key...'
	openssl genrsa 4096 > "${CA_KEY:?}" 2>/dev/null
	rm -f "${CA_CRT:?}"
fi

if [ ! -e "${CA_CRT:?}" ] || ! openssl x509 -in "${CA_CRT:?}" -noout >/dev/null 2>&1; then
	printf '%s\n' 'Generating CA certificate...'
	openssl req -new \
		-key "${CA_KEY:?}" \
		-out "${CA_CRT:?}" \
		-subj '/CN=daemon:Container daemon CA' \
		-x509 \
		-days 825
	rm -f "${SERVER_CRT:?}" "${CLIENT_CRT:?}"
fi

if [ ! -e "${SERVER_KEY:?}" ] || ! openssl rsa -check -in "${SERVER_KEY:?}" -noout >/dev/null 2>&1; then
	printf '%s\n' 'Generating server private key...'
	openssl genrsa 4096 > "${SERVER_KEY:?}" 2>/dev/null
	rm -f "${SERVER_CRT:?}"
fi

if [ ! -e "${SERVER_CRT:?}" ] || ! openssl verify -CAfile "${CA_CRT:?}" "${SERVER_CRT:?}" >/dev/null 2>&1; then
	printf '%s\n' 'Generating server certificate...'
	openssl req -new \
		-key "${SERVER_KEY:?}" \
		-out "${SERVER_CSR:?}" \
		-subj '/CN=daemon:Container daemon server'
	cat > "${SERVER_OPENSSL_CNF:?}" <<-EOF
		[ x509_exts ]
		subjectAltName = DNS:localhost,DNS:cetusguard,DNS:dockerd,IP:127.0.0.1,IP:::1
	EOF
	openssl x509 -req \
		-in "${SERVER_CSR:?}" \
		-out "${SERVER_CRT:?}" \
		-CA "${CA_CRT:?}" \
		-CAkey "${CA_KEY:?}" \
		-CAcreateserial \
		-days 825 \
		-extfile "${SERVER_OPENSSL_CNF:?}" \
		-extensions x509_exts \
		>/dev/null 2>&1
	openssl x509 -in "${SERVER_CRT:?}" -fingerprint -noout
	cp -f "${CA_CRT:?}" "${SERVER_CA_CRT:?}"
fi

if [ ! -e "${CLIENT_KEY:?}" ] || ! openssl rsa -check -in "${CLIENT_KEY:?}" -noout >/dev/null 2>&1; then
	printf '%s\n' 'Generating client private key...'
	openssl genrsa 4096 > "${CLIENT_KEY:?}" 2>/dev/null
	rm -f "${CLIENT_CRT:?}"
fi

if [ ! -e "${CLIENT_CRT:?}" ] || ! openssl verify -CAfile "${CA_CRT:?}" "${CLIENT_CRT:?}" >/dev/null 2>&1; then
	printf '%s\n' 'Generating client certificate...'
	openssl req -new \
		-key "${CLIENT_KEY:?}" \
		-out "${CLIENT_CSR:?}" \
		-subj '/CN=daemon:Container daemon client'
	cat > "${CLIENT_OPENSSL_CNF:?}" <<-EOF
		[ x509_exts ]
		extendedKeyUsage = clientAuth
	EOF
	openssl x509 -req \
		-in "${CLIENT_CSR:?}" \
		-out "${CLIENT_CRT:?}" \
		-CA "${CA_CRT:?}" \
		-CAkey "${CA_KEY:?}" \
		-CAcreateserial \
		-days 825 \
		-extfile "${CLIENT_OPENSSL_CNF:?}" \
		-extensions x509_exts \
		>/dev/null 2>&1
	openssl x509 -in "${CLIENT_CRT:?}" -fingerprint -noout
	cp -f "${CA_CRT:?}" "${CLIENT_CA_CRT:?}"
fi
