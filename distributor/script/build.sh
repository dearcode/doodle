#!/bin/bash

set -e

IFS=$'\n'

url=$1
key=$2
name=$3

framework_package="github.com/dearcode/doodle/service"

function convert_url() {
    if [[ "$url" =~ "http://" ]]
    then
        url=`echo $url|sed 's/http:\/\//git@/g'|sed 's/\//:/'|sed 's/$/.git/'`
    fi
}

function create_path() {
    base_path=`echo $url|awk -F'[@:/]' '{print "src/"$2"/"$3}'`
    rm -rf $base_path
    mkdir -p $base_path
}

function clone_source() {
    cd $base_path;
    git clone --depth=1 $url
    cd -;

    app=`basename "$url" .git`;
    base_path=$base_path/$app;

    cd $base_path;

    git_hash=`git log --pretty=format:'%H' -1`
    git_time=`git log --pretty=format:'%ct' -1`
    git_message=`git log --pretty=format:'%cn %s %b' -1`

    rm -rf .git

    cd -;
}

function generate_document() {
    export GOPATH=`pwd`
    #echo $GOPATH
    cd $base_path;
    document_file="vendor/$framework_package/generate_document.go"

    printf "package service\n\n\n" > $document_file
    printf "//init 导出的函数\n" >> $document_file
    printf "func init() { \n" >> $document_file

    for pkg in `go list ./...`
    do
        echo "package:" $pkg
        for struct in `go doc -u -cmd $pkg|awk '/^type /{print $2}'`
        do
            structKey=$pkg.$struct
            echo "struct: $structKey"
            for m in `go doc -u -cmd $structKey|awk -F'[ |(]' '/^func /{print $5}'`
            do
                methodKey=$structKey.$m
                echo "method: $methodKey" 
                comment=`go doc -u -cmd $methodKey|sed 1d`
                echo $comment
                if [ "$pkg" == "`go list`" ] 
                then
                    methodKey="$pkg/main.$struct.$m"
                fi
                echo "docExport[\"$methodKey\"] = \`$comment\`" >> $document_file
            done
        done
    done

    echo "}" >> $document_file;

    go fmt $document_file;

    cd -;
}

function create_dockerfile() {
    project=`echo $url|sed 's/.*@//'|sed 's/\.git//'|sed 's/:/\//'`;
    package_in_vendor="$project/vendor/$framework_package"
    cp Dockerfile.tpl Dockerfile
    sed -i "s#{{BASE_PATH}}#$base_path#" Dockerfile
    sed -i "s#{{PACKAGE_IN_VENDOR}}#$package_in_vendor#g" Dockerfile
    sed -i "s#{{KEY}}#$key#g" Dockerfile
    sed -i "s#{{GIT_HASH}}#$git_hash#g" Dockerfile
    sed -i "s#{{GIT_TIME}}#$git_time#g" Dockerfile
    sed -i "s#{{GIT_MESSAGE}}#$git_message#g" Dockerfile
    sed -i "s#{{PROJECT}}#$project#g" Dockerfile
}

function build() {
    version=`date -d @"$git_time" +%Y%m%d.%H%M`
    image="$project:$version"
    local src=`basename $project`
    local dest=$name
    docker build --no-cache -t $image .
    docker run -i --rm -v $PWD/bin:/base $image bash -c 'cp $GOPATH/bin/'$src' /base/'$dest 
}


convert_url

create_path

clone_source

generate_document

create_dockerfile

build

