# credit: http://www.pixelite.co.nz/article/extracting-file-folder-from-git-repository-with-full-git-history/

.PHONY: help
# Based on https://gist.github.com/rcmachado/af3db315e31383502660
## display this help text.
help:/
	$(info Available targets)
	$(info -----------------)
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		helpCommand = substr($$(pkg), 0, index($$(pkg), ":")-1); \
		if (helpMessage) { \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			gsub(/##/, "\n                                     ", helpMessage); \
		} else { \
			helpMessage = "(No documentation)"; \
		} \
		printf "%-35s -> %s\n", helpCommand, helpMessage; \
		lastLine = "" \
	} \
	{ hasComment = match(lastLine, /^## (.*)/); \
          if(hasComment) { \
            lastLine=lastLine$$0; \
	  } \
          else { \
	    lastLine = $$0 \
          } \
        }' $(MAKEFILE_LIST)


.PHONY: deps
## fetch vendor dependencies
deps:
	@echo "fetching dependencies..."
	dep ensure -v

.PHONY: build
## build all packages
build: deps
	@echo "building all packages..."
	go build ./...

.PHONY: import
## import a pkg or a file from another repository, along with the commit history
import: import-multiple-commits build

.PHONY: import-multiple-commits
import-multiple-commits:
# export the commits (as patches) from the source repo
	@echo "exporting content and commit history of pkg or file '$(pkg)' from '$(project)'..." 
	@cd $(GOPATH)/src/github.com/fabric8-services/$(project) 1>/dev/null && \
	git log --pretty=email --patch-with-stat --reverse --full-index --binary -- $(pkg) > /tmp/migrate.patch
	
	@echo "converting goimports from \"github.com/fabric8-services/$(project)\" to \"github.com/fabric8-services/fabric8-common\"..."
# replace imports of root pkg
	@$(eval SED_REGEX:="s/\"github.com\/fabric8-services\/$(project)\"/\"github.com\/fabric8-services\/fabric8-common\"/g") 
	@eval sed -i -e $(SED_REGEX) /tmp/migrate.patch
# rename imports of sub pkg
	@$(eval SED_REGEX:="s/\"github.com\/fabric8-services\/$(project)\/\([a-zA-Z0-9/]*\)\"/\"github.com\/fabric8-services\/fabric8-common\/\1\"/g")
	@sed -i -e $(SED_REGEX) /tmp/migrate.patch 
# import the commits into the target repo
	@echo "importing pkg or file '$(pkg)' with commit history into `pwd`"
	@git am /tmp/migrate.patch 

.PHONY: import-commit
## import a pkg or a file from another repository, along with the commit history
import-commit: import-single-commit build

.PHONY: import-single-commit
import-single-commit:
# export the commits (as patches) from the source repo
	@echo "exporting content and log of '$(hash)' from '$(project)'..." 
	@cd $(GOPATH)/src/github.com/fabric8-services/$(project) 1>/dev/null && \
	git show --pretty=email --patch-with-stat --reverse --full-index --binary $(hash) > /tmp/migrate.patch
	
	@echo "converting goimports from \"github.com/fabric8-services/$(project)\" to \"github.com/fabric8-services/fabric8-common\"..."
# replace imports of root pkg
	@$(eval SED_REGEX:="s/\"github.com\/fabric8-services\/$(project)\"/\"github.com\/fabric8-services\/fabric8-common\"/g") 
	@eval sed -i -e $(SED_REGEX) /tmp/migrate.patch
# rename imports of sub pkg
	@$(eval SED_REGEX:="s/\"github.com\/fabric8-services\/$(project)\/\([a-zA-Z0-9/]*\)\"/\"github.com\/fabric8-services\/fabric8-common\/\1\"/g")
	@sed -i -e $(SED_REGEX) /tmp/migrate.patch 

# import the commits into the target repo
	@echo "importing commit '$(hash)' with commit history into `pwd`"
	@git am /tmp/migrate.patch 

