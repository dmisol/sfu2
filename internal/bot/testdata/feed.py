import asyncio
import websockets


# create handler for each connection
async def handler(websocket, path):
    f = open("pcm.raw", "rb")
    x = 0
    while True:
        data = await websocket.recv()
        l = len(data)

        rd = f.read(l)
        l = len(rd)

        if l == 0:
            print("restarting file")
            f.close()
            f = open("pcm.raw", "rb")

            rd = f.read(l)
            l = len(rd)

            x = 0

        x += l
        await websocket.send(rd)
        print("sent %d %d" % (l, x))


start_server = websockets.serve(handler, "localhost", 8081)

asyncio.get_event_loop().run_until_complete(start_server)

asyncio.get_event_loop().run_forever()
