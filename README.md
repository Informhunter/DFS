# DFS

Experimental project of distributed file storage system.

Goal of the project is to create file storage system that

- Is distributed
Can keep files on many instances

- Is flexible
Can have variable system configurations with different frontends, different balance and safety polices.

- Is safe
Can replicate copies of files among several instances 

- Is scalable
Can balance load among all the instances

- Is robust
Can work even if some of the instances stop working


At the curent moment system
- Has REST-like frontend
- Distributes files among ALL its instances
- Balances load at random
- Is being configured with JSON config file
- Can't handle death of one of the instances
