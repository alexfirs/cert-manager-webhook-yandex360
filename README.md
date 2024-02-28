# Cert-Manager DNS01 webhook for the Yandex360

### Intro

cert-manager automates the management and issuance of TLS certificates in Kubernetes clusters. It ensures that certificates are valid and updates them when necessary.

A certificate authority resource, such as ClusterIssuer, must be declared in the cluster to start the certificate issuance procedure. It is used to generate signed certificates by honoring certificate signing requests.

For some DNS providers, there are no predefined CusterIssuer resources. Fortunately, cert-manager allows you to write your own ClusterIssuer.

This solver allows you to use cert-manager with the Yandex360 API. Documentation on the Yandex360 API is available [here](https://yandex.ru/dev/api360/doc/ref/DomainDNSService.html).

Yandex360 allows to have multiple organizations per account and each organization may have multiple domains, so it will be required to create an issuer per organization. You also will need organization id (can be found on the left bottom of a web page in a browser when organization is selected on https://admin.yandex.ru)


# Usage

### Preparation

You must [get an api token with](https://yandex.ru/dev/api360/doc/concepts/access.html)
```
directory:manage_dns 
```
permission. 

1. Create app with directory:manage_dns permission. For the redirect url use placeholder - https://oauth.yandex.ru/verification_code, this page will show token on successful login
2. Remember ClientId from app page and navigate to https://oauth.yandex.ru/authorize?response_type=token&client_id=<CLIENT_ID>
3. After authorization save received token. Token valid for one year
4. (optional) check if token works executing
```
curl https://api360.yandex.net/directory/v1/org/<ORG_ID>/domains/<DOMAIN>/dns --header 'Authorization: OAuth <AUTH_TOKEN>'
```

### Install cert-manager (*optional step*)

**ATTENTION!** Yandex360 seems to update dns entries **VERY** slow, so in order to make it working you will need to update cert-manager (or create separate instance in separate namespace) with 
```yaml
- --dns01-check-retry-period=600s
- --dns01-recursive-nameservers-only 
- --dns01-recursive-nameservers=8.8.8.8:53,1.1.1.1:53
```
arguments for controller deployment

**ATTENTION!** You should not delete the cert-manager if you are already using it.

Use the following command from the [official documentation](https://cert-manager.io/docs/installation/) to install cert-manager in your Kubernetes cluster:

```shell
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/VERSION/cert-manager.yaml
```
*  where `VERSION` is necessary version (for example, v1.14.3 )

### Install the webhook
Start with 
```shell
git clone https://github.com/alexfirs/cert-manager-webhook-yandex360.git
```
Now you have 2 options - use precompiled image from docker hub or compile it by yourselv

#### if you want to build from source
```shell
make build
```


Edit the `deploy/webhook-yandex360/values.yaml` file in the cloned repository and enter the appropriate values in the fields, primarly group name. If you built it manually - provide correct image name
```yaml
issuer:
  image: alexfirs/cert-bot-webhook-yandex360:1.0.0
```

You must also specify your namespace with the `cert-manager`.

```yaml
certManager:
  namespace: my-namespace-cert-manager
  serviceAccountName: cert-manager
```



Next, run the following commands for the install webhook.

```shell
cd cert-manager-webhook-yandex360
helm install -n my-namespace-cert-manager webhook-yandex360 ./deploy/webhook-yandex360
```

### Create a ClusterIssuer

Create the `ClusterIssuer.yaml` file with the following contents (make sure you update the dnsZones and group name. You will need a cluster issuer per organization): 
```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging-with-yandex360
spec:
  acme:
    # prod : https://acme-v02.api.letsencrypt.org/directory
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: your@email.com #this needs to be updated
    privateKeySecretRef:
      name: letsencrypt-staging
    solvers:
    - selector:
        dnsZones:
        - 'alexfirs.ru' # afirs.ru and *.afirs.ru
      dns01:
        webhook:
          config:
            apiTokenSecretRef:
              name: yandex360-secret
              key: token
            organizationId: 123456789
            endpoint: "https://api360.yandex.net"
          groupName: acme.alexfirs.ru
          solverName: yandex360-dns-solver
```
and create the resource:

```shell
kubectl create -f ClusterIssuer.yaml
```

#### Token

You have to provide a `token` for the webhook so that it can access the HTTP API.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: yandex360-secret
  namespace: cert-manager
type: Opaque
stringData:
  token: "<TOKEN>"
```
and create a resource
```shell
kubectl create -f Secret.yaml
```

### Create a certificate

Create the `certificate.yaml` file with the following contents:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: acmetest-star-afirs-ru
spec:
  secretName: star-afirs-ru-tls
  duration: 2160h # 90d
  renewBefore: 360h # 15d
  dnsNames:
  - '*.afirs.ru'
  issuerRef:
    name: letsencrypt-staging-with-yandex360
    kind: ClusterIssuer
```

## Tests

You can run the webhook test suite with:

```bash
$ TEST_ZONE_NAME=example.com. make test
```

# Community

Please feel free to contact me if you have any questions - notffirk@gmail.com


# License

Apache License 2.0, see [LICENSE](LICENSE).

# Thanks
Thanks for inspiration
```
https://github.com/boryashkin/cert-manager-webhook-beget
https://github.com/flant/cert-manager-webhook-regru
```