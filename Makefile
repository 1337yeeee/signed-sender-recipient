.PHONY: keys

KEYS_DIR := data/keys
APP_PRIVATE_KEY := $(KEYS_DIR)/private.pem
APP_PUBLIC_KEY := $(KEYS_DIR)/public.pem

keys:
	mkdir -p $(KEYS_DIR)
	openssl ecparam -name prime256v1 -genkey -noout -out $(APP_PRIVATE_KEY)
	openssl ec -in $(APP_PRIVATE_KEY) -pubout -out $(APP_PUBLIC_KEY)
