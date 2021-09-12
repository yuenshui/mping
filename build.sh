#!/usr/bin/env bash

version=`cat version`
timeTag="-X 'main.BuildTime=$(date '+%Y-%m-%d %H:%M:%S')'"
branchFlag="-X main.GitBranch=$(git name-rev --name-only HEAD)"
commitFlag="-X main.CommitId=$(git rev-parse --short HEAD)"
goVersion=`go version | awk '{print $3}'`
goVersionFlag="-X 'main.GoVersion=${goVersion}'"
staticTag="-extldflags '-static'" #关闭符号链接
ldflags="-s -w ${staticTag} ${timeTag} ${branchFlag} ${commitFlag} ${goVersionFlag}"
function build() {
  FILENAME="${1}-${2}-${version}-mping"
  if [ $1 == "windows" ] ;then
    FILENAME="${FILENAME}.exe"
  fi
  CGO_ENABLED=0 GOOS=${1} GOARCH=${2} go build -o bin/${FILENAME} -ldflags "${ldflags}" main.go > runBuild.log 2>&1
  if [ $? == 0 ] ;then
    echo $FILENAME
  else
    echo $FILENAME "error"
  fi
}

goBuild=$(go tool dist list)
strArr=(${goBuild//,/ })  
for (( i=0; i<${#strArr[@]}; i++ )); do
  LINEARR=(${strArr[i]//\// })
  build ${LINEARR[0]} ${LINEARR[1]}
done 
