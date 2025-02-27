package lock

import (
	"time"

	"6.5840/kvsrv1/rpc"
	kvtest "6.5840/kvtest1"
)

type Lock struct {
	// IKVClerk is a go interface for k/v clerks: the interface hides
	// the specific Clerk type of ck but promises that ck supports
	// Put and Get.  The tester passes the clerk in when calling
	// MakeLock().
	ck kvtest.IKVClerk
	// You may add code here
	key      string
	clientID string
}

// The tester calls MakeLock() and passes in a k/v clerk; your code can
// perform a Put or Get by calling lk.ck.Put() or lk.ck.Get().
//
// Use l as the key to store the "lock state" (you would have to decide
// precisely what the lock state is).
func MakeLock(ck kvtest.IKVClerk, l string) *Lock {
	lk := &Lock{
		ck:       ck,
		key:      l,
		clientID: kvtest.RandValue(8),
	}
	return lk
}

func (lk *Lock) Acquire() {
	for {
		lockClient, remoteVersion, err := lk.ck.Get(lk.key)
		if err == rpc.OK {
			if lockClient == lk.clientID {
				// We already have the lock.
				return
			} else if lockClient == "" {
				// No one has the lock.
				err = lk.ck.Put(lk.key, lk.clientID, remoteVersion)
				if err == rpc.OK {
					return
				} else {
					// Lock information is modified by other client. Retry.
					continue
				}
			} else {
				// Other client has the lock. Wait for it to release.
				time.Sleep(10 * time.Millisecond)
			}
		} else {
			// Lock is not created yet. Create it.
			err = lk.ck.Put(lk.key, lk.clientID, 0)
			if err == rpc.OK {
				return
			} else {
				// Lock is created by other client. Retry.
				continue
			}
		}
	}
}

func (lk *Lock) Release() {
	for {
		lockClient, remoteVersion, err := lk.ck.Get(lk.key)
		if err != rpc.OK {
			// Lock is not created yet. Nothing to release.
			return
		}
		if lockClient == lk.clientID {
			// We have the lock. Release it.
			err = lk.ck.Put(lk.key, "", remoteVersion)
			if err == rpc.OK {
				return
			}
		} else {
			// We don't have the lock. Nothing to release.
			return
		}
	}
}
