
encrypt_secret:
	 @openssl aes-256-cbc -e -in secret.json -out client_secret.json.enc -k $(METRICS_DECODE_SECRET_KEY)

decrypt_secret:
	@openssl aes-256-cbc -d -in client_secret.json.enc -out client_secret.json  -k $(METRICS_DECODE_SECRET_KEY)

run: decrypt_secret
	go run *.go
	