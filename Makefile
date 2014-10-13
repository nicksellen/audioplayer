
.PHONY: jsx

jsx:
	jsx --harmony \
			--cache-dir /tmp/jsx-cache \
			--watch jsx assets/build
