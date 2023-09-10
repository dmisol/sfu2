.MAIN: sfu

sfu:
	docker stop sfu2 || true
	docker rm sfu2 || true

	docker build -t sfu2 -f sfu2.dockerfile .
	docker run \
		--name sfu2 \
		--network="host" \
		-e AUDIT_LEVEL=1 \
		-v /tmp:/tmp \
		-v /ftars:/ftars \
		sfu2 &

