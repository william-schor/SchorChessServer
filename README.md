# SchorChessServer

## Server Thread Structure: 

Connection thread: 
* accepts connections, hands them off to client threads

Client Threads:
 * Add self to "online" list 
    * Consider how to remove failed connections

* Iterate over pairing list
    * pair if match is available
    * add to list if not

* How to add self to list:
    * add channel to list, wait for it to fill
* How to pair: 
    * Flip coin for color
    * Tell other thread its color
    * Take channel off list 
* How to play:
    * Send moves to one another via channel

