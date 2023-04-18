import wave
import asyncio
import websockets


# create handler for each connection
async def handler(websocket, path):
    sampleRate = 48000.0  # hertz
    obj = wave.open('sound.wav', 'w')
    obj.setnchannels(1)  # mono
    obj.setsampwidth(2)
    obj.setframerate(sampleRate)
    while True:
        data = await websocket.recv()
        obj.writeframesraw(data)


start_server = websockets.serve(handler, "localhost", 8081)


asyncio.get_event_loop().run_until_complete(start_server)

asyncio.get_event_loop().run_forever()
