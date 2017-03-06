#!/usr/bin/env bash

APPLICATION=mafio

nohup ./${APPLICATION} -config=./agent.conf  &
ps aux|grep ${APPLICATION}|grep -v grep
