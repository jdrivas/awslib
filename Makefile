lib := awslib
repo := github.com/jdrivas/$(lib)

help:
	@echo release \# push master branch to github and then do a local go update.

release:
	go build
	git push
	go get -u $(repo)
