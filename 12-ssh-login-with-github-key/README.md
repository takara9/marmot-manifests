# GitHubに登録した公開鍵でSSHログインできるサーバー

GitHubで公開しているssh公開鍵を使って、ログインできるようにします。


```yaml
    auth:
        url: https://github.com/takara9.keys
        users:
            - ubuntu
```