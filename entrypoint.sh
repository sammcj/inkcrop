#!/usr/bin/env bash

# A simple entrypoint for the container that:
# - Starts the app (inkcrop) in daemon mode and backgrounds but captures any crashes and logs them to stderr before exiting
# - Starts nginx as a daemon and backgrounds it but captures any crashes and logs them to stderr before exiting

inkcropArgs=${INKCROP_ARGS:-('-daemon' 'true' '-dither' 'true' '-input' '/input' '-output' '/output')}
nginxArgs=${NGINX_ARGS:-('-g' 'daemon off;' '-c' '/etc/nginx/nginx.conf')}

# Start the app
./inkcrop "$inkcropArgs" &
# Capture the PID of the app
appPid=$!

# Start nginx
nginx "$nginxArgs" &
# Capture the PID of nginx
nginxPid=$!

echo "Running inkcrop with args: ${inkcropArgs}"
echo "Running nginx with args: ${nginxArgs}"
echo "App PID: ${appPid}"
echo "Nginx PID: ${nginxPid}"

# Wait for either the app or nginx to exit
wait -n

# If the app exited, kill nginx
if [ $? -eq $appPid ]; then
  kill $nginxPid
  echo "App exited with code $?" >&2
fi

# If nginx exited, kill the app
if [ $? -eq $nginxPid ]; then
  kill $appPid
  echo "Nginx exited with code $?" >&2
fi

# Exit with the exit code of the process that exited
exit $?
