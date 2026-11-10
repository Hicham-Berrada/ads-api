# Notes

I did the items in order, one commit each, so the diffs read one after the other. The layering was already there (http → usecase → dao) and I kept it that way: handlers only deal with HTTP, usecases hold the rules, the DAO owns the SQL. I didn't touch `/users`.

`go test ./...` was already red when I started (inverted assertion in the DAO mock test), so the first commit is just that fix. I also cleaned up two stale comments next to code I was already editing, that's the `chore` one.

## 1. GetAdByID

The usecase was passing `0` instead of the id. The tests didn't see it because they matched on `mock.Anything`, so the mock answered happily. They check the real id now.

## 2. Delete

`DELETE /ads/{adID}`, owner only, 204. Single query, `DELETE ... WHERE id = $1 AND user_id = $2`. Someone else's ad gets a 404 like a missing one — I don't see why we would tell the caller that the id exists.

## 3. Username

Read from the users service with the client you provided. If that call fails I still return the ad, without `username`, and log a warning. Reads are the heavy side of this service, they shouldn't start failing because another service is having a bad day. Next step would be a small cache in front of the client; the 2s default timeout is also probably too generous for a field we can live without.

## 4. price as a float

Not done, on purpose. `19.99 * 100` in float64 gives `1998.9999999999998`, and renaming the field breaks the mobile apps already in the stores. Before doing anything I'd want to know what the need is. If it's only for display, clients can format the cents. Otherwise I'd add a second field next to `price_cents` and deprecate the first one, keeping cents in the database.

## 5. Bulk import

Each ad was saved on its own, so one failure left the previous ones in, and the retry created them again. It's one transaction now.

What I didn't solve: the import succeeds but the response never reaches the client, so it retries anyway. That needs an idempotency key coming from the client, it's a discussion with the client teams.

## 6. Quota

5 ads per rolling 24h on `POST /ads`, 429 above that.

Publications get their own table (`ad_publications`). The import doesn't write there and deleting an ad doesn't remove the row, so you can't play with create/delete to get around the limit.

Counting and inserting in the same transaction isn't enough on its own: two requests can both count 4. I take a Postgres advisory lock on the user before counting, it goes away with the transaction. There's an integration test firing 10 concurrent requests (`go test -tags integration ./...`).

Something for product: the import isn't limited, so it is an obvious way around the quota.

## Loose ends

- `GET /ads` has no pagination and no ordering.
- Nothing validates the create payload (empty title, negative price).
- 500s aren't logged, and the error responses have no body.
- A non-UUID `Authorization` header ends up as a 500 on writes.

With another evening I'd do the pagination and the idempotency key. Happy to walk through any of it.
