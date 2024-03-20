# fair-router is a POC for a fair and efficient request routing algorithm.

If you have HTTP requests that you want to distribute among a set of workers which have limited heterogeneous processing capacity, you can use `fair-router` to do so. It is a simple and efficient algorithm that ensures that workers activity keep the router updated with how much processing capacity they have left.

