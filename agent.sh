#!/usr/bin/env bash

APPLICATION=mafio

nohup ./${APPLICATION} -config=./base.conf  &
ps aux|grep ${APPLICATION}|grep -v grep
