SOURCE=$(shell find . -iname "*.go")

covidrecord: $(SOURCE)
	go build -o covidreport *.go

.PHONY=deploy
deploy: sync
	gcloud app deploy --quiet

# sync static assets to gcp
.PHONY=sync
sync:
	gsutil -m rsync -r static gs://covidrecord-static-assets/
	gsutil cors set cors.json gs://covidrecord-static-assets

# apply license headers to all golang files
.PHONY=license_headers
license_headers:
	for f in **/*.go; do \
		cat .license_header $$f > $$f.new ; \
		mv $$f.new $$f ; \
	done
