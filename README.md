# ziti-livekit-example
This is a working example of livekit runing behind openziti. zitified livekit sdk. publisher publishes videostream(a red triangle) to a room and subscriber container receives that stream.

To use zitified livekit sdk, check out the code in `publisher` and `subscriber` directories.

# Run
```bash
./init.sh
docker compose pull
docker compose build
docker compose up
```
wait till everything initializes(should see error of `publisher-app`/`subscriber-app` about missing configs)

Open another terminal windows
```bash
./install.sh
```
After this there should be admin1 identity in `./store/admin1.json`, add it to your local ziti tunneler(linux):
```bash
docker compose cp ziti-edge-router:/persistent/admin1.json /opt/openziti/etc/identities
sudo systemctl restart ziti-edge-tunnel.service
```

Now go back to first terminal, Ctrl+C and then run `docker compose up` again. if you have a running tuneler on your host with admin1 identity added, you should see `publisher-app` and `subscriber-app` sending and receiving udp packets in logs.

# uninstall
```bash
./uninstall.sh
```