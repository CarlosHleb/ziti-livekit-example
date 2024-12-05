# ziti-livekit-example
This is a non working example(you need to run a openziti tunneler on the host for it to work.)

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