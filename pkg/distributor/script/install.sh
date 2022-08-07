#!/bin/bash


project=$1
pid=$2
etcd=$3


function project_init() {
    mkdir -p $project/bin
    mkdir -p $project/logs
    mkdir -p $project/config
    mv $project.tar.gz $project
}


function send_close_signal() {
    if [ $pid == 0 ] 
    then
        return
    fi

    pgrep_pid=`pgrep $project|grep $pid`

    if [[ "$pgrep_pid" == "$pid" ]]
    then
        echo "send close signal $pid"
        kill -s SIGUSR1 $pid
    fi
}


function extract_project() {
    tar xzf $project.tar.gz -C bin
}

function daemon_start() {
    nohup ./bin/$project -etcd $etcd -h : >> daemon.log 2>&1 &
}



project_init

cd $project

extract_project

send_close_signal

daemon_start

cd -
