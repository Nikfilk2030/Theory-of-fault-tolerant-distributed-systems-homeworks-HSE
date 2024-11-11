# Theory-of-fault-tolerant-distributed-systems-homeworks-HSE
Higher School of Economics, Faculty of Computer Science, 2024

## Task 1

Write a system that calculates the integral of some function.

The master (client) finds worker nodes (servers) via IP broadcast - it sends out a start message to all subnet addresses, to which worker nodes listening on their TCP ports reply. Then each worker node is given a segment, it calculates an integral on it and sends the answer to the master. The master adds up the servers' responses and gets the final result.

### Requirements:

- if after job distribution the servers become unavailable (shutdown / network breakdown), but at least one server is available, the program detects this, distributes jobs to available servers instead of the disconnected ones and gives the correct answer

- if the unavailable server reappears in the network and tries to send a response, this does not cause an error, in particular, the result of the corresponding segment will not be counted twice.

- if an unavailable server appears in the network, the wizard should be able to send new tasks to it (for example, some other server is down).

Please do the task in pure C, using the network sockets API. It is best to do it on a UNIX-like system, although sockets are generally similar on Windows.

Be sure to show the work of the program with full/partial packet loss, duplication, delays. I recommend tc or iptables utility.

Literature:

Stevens W. R. “Network Application Development”, ch. 2, 3, 4, 5, 7
