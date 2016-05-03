# Buddy Broker

## What is this?

Cloud Foundry recently added private service brokers! This is great but that broker is limited to one space. Buddy to the rescue! Every service broker needs a buddy to hang out with. This app adds suffix to guids/names so the same broker can be used for multiple spaces

## Deploying

```
$ cf push buddy-broker --no-start
$ cf set-env buddy-broker BACKEND_BROKER ${broker_url}
$ cf start buddy-broker
```

That's it!

### Registering broker

```
$ cf create-service-broker buddy-space1 ${username} ${password} ${buddy_url}/space1
```

This will add suffix to your service broker ids/name. ie. redis-space1.

**Note** Username and password of broker is transparently passed to broker

