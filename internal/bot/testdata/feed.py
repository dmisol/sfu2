import asyncio
import websockets


# create handler for each connection
async def handler(websocket, path):
    f = open("48k.raw", "rb")
    x = 0
    while True:
        data = await websocket.recv()
        li = len(data)

        rd = f.read(3*li)
        lo = len(rd)	# data is supplied at 16k, feeded at 48k

        if lo != 3*li:
            print("restarting file")
            f.close()
            f = open("48k.raw", "rb")

            rd = f.read(3*li)
            lo = len(rd)

            x = 0

        x += lo
        await websocket.send(rd)
        print("sent %d %d" % (lo, x))


start_server = websockets.serve(handler, "localhost", 8081)

asyncio.get_event_loop().run_until_complete(start_server)

asyncio.get_event_loop().run_forever()
