BUILD_DATE=`date +'%Y%m%d'`
GIT_REVISON=`git rev-parse HEAD`
go get -ldflags="-X main.buildDate=$BUILD_DATE -X main.gitVersion=$GIT_REVISON"
