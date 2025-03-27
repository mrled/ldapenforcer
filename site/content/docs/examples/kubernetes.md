+++
title = "Kubernetes"
+++

A real example from my homelab.

## Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ldapenforcer
  namespace: directory
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ldapenforcer
  template:
    metadata:
      labels:
        app: ldapenforcer
    spec:
      containers:
        - name: enforcer
          image: ghcr.io/mrled/ldapenforcer:0.1.3
          args:
            - "--config"
            - "/etc/ldapenforcer/ldapenforcer.toml"
            - "sync"
            - "--poll"
            - "--log-level"
            - "DEBUG"
            - "--ldap-log-level"
            - "DEBUG"
          volumeMounts:
            - name: ldapenforcer-cm
              mountPath: /etc/ldapenforcer
            - name: dirsrv-tls-ca
              mountPath: "/data/tls/ca"
            - name: dirsrv-env-secret
              mountPath: /etc/dirsrv/env-secret
          securityContext:
            runAsUser: 389
            runAsGroup: 389

      volumes:
        - name: dirsrv-tls-ca
          configMap:
            name: kubernasty-ca-root-cert
        - name: dirsrv-env-secret
          secret:
            secretName: dirsrv-env
        - name: ldapenforcer-cm
          configMap:
            name: ldapenforcer
```

## ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ldapenforcer
  namespace: directory
data:
  ldapenforcer.toml: |+
    [ldapenforcer]
    uri = "ldaps://dirsrv.directory.svc.cluster.local:636"
    bind_dn = "cn=Directory Manager"
    password_file = "/etc/dirsrv/env-secret/DS_DM_PASSWORD"
    ca_cert_file = "/data/tls/ca/ca.crt"

    enforced_people_ou = "ou=enforced,ou=people,dc=micahrl,dc=me"
    enforced_svcacct_ou = "ou=enforced,ou=services,dc=micahrl,dc=me"
    enforced_group_ou = "ou=enforced,ou=groups,dc=micahrl,dc=me"

    poll_config_interval = "10s"
    poll_ldap_interval = "1h"

    log_level = "DEBUG"
    ldap_log_level = "DEBUG"

    includes = [
      "svcaccts.toml",
      "people.toml",
      "groups.toml",
    ]

  svcaccts.toml: |+
    [ldapenforcer.svcacct.authenticator]
    cn = "Authenticator"
    description = "A service account for authenticating users"

    [ldapenforcer.svcacct.ldapAccountManager]
    cn = "LDAP Account Manager"
    description = "A service account for managing LDAP accounts"

    [ldapenforcer.svcacct.authelia]
    cn = "Authelia"
    description = "A service account for Authelia"

  people.toml: |+
    [ldapenforcer.person.mrladmin]
    cn = "Micah R Ledbetter (Admin)"
    givenName = "Micah"
    sn = "Ledbetter"
    mail = "mrladmin@micahrl.me"
    posix = [10420, 10100]

    [ldapenforcer.person.micahrl]
    cn = "Micah R Ledbetter"
    givenName = "Micah"
    sn = "Ledbetter"
    mail = "me@micahrl.com"
    posix = [10069, 10101]

  groups.toml: |+
    [ldapenforcer.group.patricii]
    description = "Accounts with administrative privileges"
    posixGidNumber = 10100
    people = ["mrladmin"]

    [ldapenforcer.group.proletarii]
    description = "Regular user accounts"
    posixGidNumber = 10101
    people = ["micahrl"]

    [ldapenforcer.group.servi]
    description = "Service accounts"
    posixGidNumber = 10102
    svcaccts = ["authelia", "authenticator", "ldapAccountManager"]

    [ldapenforcer.group.totalgits]
    description = "Users that can log in to the Git server"
    people = ["mrladmin", "micahrl"]

    [ldapenforcer.group.argowf-users]
    description = "Users that can log in to the Argo Workflows server"
    groups = ["proletarii"]

    [ldapenforcer.group.argowf-admins]
    description = "Users that can administer the Argo Workflows server"
    groups = ["patricii"]

    [ldapenforcer.group.grafana-users]
    description = "Users that can log in to the Grafana server"
    groups = ["proletarii"]

    [ldapenforcer.group.grafana-admins]
    description = "Users that can administer the Grafana server"
    groups = ["patricii"]
```

## Not shown: CA resource

The cluster-wide certificate authority has signed the LDAP server's TLS certificates.
The CA cert is found in the `kubernasty-ca-root-cert` configmap,
and mounted into the Deployment pod.

## Not shown: dirsrv-env Secret

The drsrv-env Secret resource contains secret environment variables for the deployment of
[389 Directory Server](https://port389.org),
including the Directory Manager password as `DS_DM_PASSWORD`.
This user will be used to sync the config to the LDAP server.
