package server

import (
	"errors"
	"fmt"
)

// Client represents a connected Faye Client, each has an ID negotiated during handshake, a write channel tied to their network connection
// and a list of subscriptions(faye channels) that have been subscribed to by the client.
type Client struct {
	ClientID     string
	WriteChannel chan []byte
	ClientSubs   []string
}

// isSubscribed checks the subscription status for a given subscription name
func (c *Client) isSubscribed(sub string) bool {
	for _, clientSub := range c.ClientSubs {
		if clientSub == sub {
			return true
		}
	}
	return false
}

// subscriptionClientIndex finds the index for the given clientID in the subscriptions or returns -1 if not found
func (s *Server) subscriptionClientIndex(subscriptions []Client, clientID string) int {
	for i, c := range subscriptions {
		if c.ClientID == clientID {
			return i
		}
	}
	return -1
}

// removeSubFromClient removes a Subscription from a given client
func (s *Server) removeSubFromClient(client Client, sub string) Client {
	for i, clientSub := range client.ClientSubs {
		if clientSub == sub {
			client.ClientSubs = append(client.ClientSubs[:i], client.ClientSubs[i+1:]...)
			return client
		}
	}
	return client
}

// removeClientFromSubscription removes the given clientID from the subscription
func (s *Server) removeClientFromSubscription(clientID, subscription string) bool {
	fmt.Println("Remove Client From Subscription: ", subscription)

	// grab the client subscriptions array for the channel
	s.SubMutex.Lock()
	defer s.SubMutex.Unlock()

	subs, ok := s.Subscriptions[subscription]

	if !ok {
		return false
	}

	index := s.subscriptionClientIndex(subs, clientID)

	if index >= 0 {
		s.Subscriptions[subscription] = append(subs[:index], subs[index+1:]...)
	} else {
		return false
	}

	// remove sub from client subs list
	s.Clients[clientID] = s.removeSubFromClient(s.Clients[clientID], subscription)

	return true
}

// addClientFromSubscription adds the given clientID to the subscription
func (s *Server) addClientToSubscription(clientID, subscription string, c chan []byte) bool {
	fmt.Println("Add Client to Subscription: ", subscription)

	// Add client to server list if it is not present
	client := s.addClientToServer(clientID, subscription, c)

	// add the client as a subscriber to the channel if it is not already one
	s.SubMutex.Lock()
	defer s.SubMutex.Unlock()
	subs, cok := s.Subscriptions[subscription]
	if !cok {
		s.Subscriptions[subscription] = []Client{}
	}

	index := s.subscriptionClientIndex(subs, clientID)

	fmt.Println("Subs: ", s.Subscriptions, "count: ", len(s.Subscriptions[subscription]))

	if index < 0 {
		s.Subscriptions[subscription] = append(subs, *client)
		return true
	}

	return false
}

// client management

// UpdateClientChannel updates the write channel for the given clientID with the provided channel
func (s *Server) UpdateClientChannel(clientID string, c chan []byte) bool {
	fmt.Println("update client for channel: clientID: ", clientID)
	s.ClientMutex.Lock()
	defer s.ClientMutex.Unlock()
	client, ok := s.Clients[clientID]
	if !ok {
		client = Client{clientID, c, []string{}}
		s.Clients[clientID] = client
		return true
	}

	client.WriteChannel = c
	s.Clients[clientID] = client

	return true
}

// addClientToServer Add Client to server only if the client is not already present
func (s *Server) addClientToServer(clientID, subscription string, c chan []byte) *Client {
	fmt.Println("Add client: ", clientID)

	s.ClientMutex.Lock()
	defer s.ClientMutex.Unlock()
	client, ok := s.Clients[clientID]
	if !ok {
		client = Client{clientID, c, []string{}}
		s.Clients[clientID] = client
	}

	fmt.Println("Client subs: ", len(client.ClientSubs), " | ", client.ClientSubs)

	// add the subscription to the client subs list
	if !client.isSubscribed(subscription) {
		fmt.Println("Client not subscribed")
		client.ClientSubs = append(client.ClientSubs, subscription)
		s.Clients[clientID] = client
		fmt.Println("Client sub count: ", len(client.ClientSubs))
	} else {
		fmt.Println("Client already subscribed")
	}

	return &client
}

// removeClientFromServer temoves the Client from the server and unsubscribe from any subscriptions
func (s *Server) removeClientFromServer(clientID string) error {
	fmt.Println("Remove client: ", clientID)

	s.ClientMutex.Lock()
	defer s.ClientMutex.Unlock()

	client, ok := s.Clients[clientID]
	if !ok {
		return errors.New("Error removing client")
	}

	// clear any subscriptions
	for _, sub := range client.ClientSubs {
		fmt.Println("Remove sub: ", sub)
		if s.removeClientFromSubscription(client.ClientID, sub) {
			fmt.Println("Removed sub!")
		} else {
			fmt.Println("Failed to remove sub.")
		}
	}

	// remove the client from the server
	delete(s.Clients, clientID)

	return nil
}
