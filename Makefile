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
