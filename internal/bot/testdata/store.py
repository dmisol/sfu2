
import asyncio
import websockets


# create handler for each connection
async def handler(websocket, path):
    f = open("pcm.raw", "wb")
    while True:
        data = await websocket.recv()
        f.write(data)

start_server = websockets.serve(handler, "localhost", 8081)

asyncio.get_event_loop().run_until_complete(start_server)

asyncio.get_event_loop().run_forever()
