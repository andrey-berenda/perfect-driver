#!/bin/bash

function get_pid() {
    pid=$(pidof /home/ec2-user/bot-running)
}
get_pid
old_pid=$pid

sudo systemctl start bot.service
sudo kill -SIGINT "$pid"

while [[ "$pid" == "$old_pid" ]]; do
  get_pid
  sleep 1;
done
sudo sudo systemctl status bot.service
