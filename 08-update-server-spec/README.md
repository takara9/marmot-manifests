# サーバーの変更

ラベルとコメントを変更できます。また、CPUとメモリも条件付きで変更できます。

仮想マシンとして、CPU、メモリの量を変更することができます。制限をかけていません。
ハードウェアのリソース量を超えるCPUとメモリ量をセットすることもできます。
また、OSが起動できないメモリ量に減らすことも可能です。


(注意)　CPUやメモリを変更すると、仮想マシンが不安定化して、起動できなくなる恐れがあります。
**この機能の使用は避けて**、サーバーの再作成を実施してください。
また、NIC、ディスクの増設、撤去は実施できません。それが必要な時は、サーバーを再作成してください。



```console
# 初期状態の確認
$ mactl get srv
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---

# サーバーのデプロイ
$ mactl create -f server-1-cpu-x1.yaml 
リソースの作成要求が受け入れられました。ID: 37341

# デプロイ完了をチェック
$ mactl get server
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server-1         ws1           RUNNING       1    1024     192.168.1.20     host-bridge      50s

# サーバーのCPU数増加
$ mactl apply -f server-1-cpu-x2.yaml 
サーバー server-1 の spec 変更には停止と再起動を伴います。続行しますか？ (Y/N): Y
リソースが更新されました。ID: 37341

# 変更結果の確認
$ mactl get server
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server-1         ws1           RUNNING       2    1024     192.168.1.20     host-bridge      3m

# メモリ量の増加とCPU数減量の実施
$ mactl apply -f server-1-mem-x2.yaml 
サーバー server-1 の spec 変更には停止と再起動を伴います。続行しますか？ (Y/N): Y
リソースが更新されました。ID: 37341

# 
$ mactl get server
NAME             NODE          STATUS        CPU  RAM(MB)  IP-ADDRESS       NETWORK          AGE
----             ----          ------        ---  -------  ----------       -------          ---
server-1         ws1           RUNNING       1    2048     192.168.1.20     host-bridge      4m

```




（注意）
サーバーの構成を変更すると、sshd が起動しなくなります。
構成変更後は、コンソールからログインして、`ssh-keygen -A`を実行して、ホストの鍵を作り直してください。

```
$ mactl console server-1
＜中略＞
# ssh-keygen -A
# systemctl restart ssh.service
```