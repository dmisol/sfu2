let recButton = document.createElement("button");
recButton.style.width = "120px";
recButton.style.height = "60px";
recButton.innerHTML = "Record";
recButton.style.display = "none";
recButton.onclick = rec;
document.getElementById("buttons").appendChild(recButton);

let playButton = document.createElement("button");
playButton.style.width = "120px";
playButton.style.height = "60px";
playButton.style.display = "none";
playButton.innerHTML = "Play";
playButton.onclick = play;
document.getElementById("buttons").appendChild(playButton);



let stopButton =document.createElement("button");
stopButton.style.width = "120px";
stopButton.style.height = "60px";
stopButton.style.display = "none";
stopButton.innerHTML = "Stop";
stopButton.onclick = stop;
document.getElementById("buttons").appendChild(stopButton);

const parentElement = document.getElementById("body");


const flxCount = 2;
const engines = [];
const flxLinks = ["/file/flexatars/flx1.p","/file/flexatars/flx2.p","/file/flexatars/flx3.p"];
(async function () {


for (let i = 0; i < flxLinks.length; i++) {


    const flxCanvas = document.createElement("canvas");
    flxCanvas.width = 240;
    flxCanvas.height = 320;
    flxCanvas.style.border = "2px solid";
    parentElement.appendChild(flxCanvas);
            aBuffer = await fetch(flxLinks[i])
            .then(response => {
                if (!response.ok) {
                  throw new Error('Network response was not ok');
                }
                return response.arrayBuffer();
            });
//            .then(arrayBuffer => {
//
//
//            })
//            .catch(error => {
//                console.error('Fetch error:', error);
//            });
    const rEngine = await makeFlexatar(flxCanvas,aBuffer)


    engines.push(rEngine)
}

})();
const speechNN = new SpeechNN();
speechNN.awaitResources().then( () => {
    recButton.style.display =  'inline-block';
//    stopButton.style.display = 'inline-block';
} );

var audioContext = new (window.AudioContext || window.webkitAudioContext)();


var processorNode;
var micSrc;
if (navigator.mediaDevices && navigator.mediaDevices.getUserMedia) {
  console.log("getUserMedia supported.");
  navigator.mediaDevices
    .getUserMedia({audio: true,},)
    .then((stream) => {
        micSrc = audioContext.createMediaStreamSource(stream);
        audioContext.audioWorklet.addModule("audio_processor.js").then(() => {

            processorNode = new AudioWorkletNode(audioContext,"my-audio-processor",);
            micSrc.connect(processorNode);
        });

    })

    // Error callback
    .catch((err) => {
      console.error(`The following getUserMedia error occurred: ${err}`);
    });
} else {
  console.log("getUserMedia not supported on your browser!");
}

var pcmBufferRec = [];

const animCalc = new AnimCalc(speechNN);
var animRecorded = []
function rec() {
    animRecorded = [];
    pcmBufferRec = [];
    recButton.style.display = 'none';
    stopButton.style.display = 'inline-block';
    playButton.style.display = 'none';

    audioContext.resume();
    processorNode.port.postMessage(true);
    processorNode.port.onmessage = (event) => {
        if (event.data){
            const speechState = animCalc.getAnim(event.data)
            pcmBufferRec.push(event.data);
        animRecorded.push(speechState);
        for (const eng of engines){
            eng.speechState = speechState;
        }
        }
    }
}
var recordedAudioBuffer;
function stop() {
    audioContext.resume();
    processorNode.port.postMessage(false);
    const pcmBuffer = concatenateFloat32Arrays(pcmBufferRec);

    const sampleRate = audioContext.sampleRate;
    const audioBuffer = audioContext.createBuffer(1, pcmBuffer.length, sampleRate);
    audioBuffer.copyToChannel(pcmBuffer, 0);
    recordedAudioBuffer = audioBuffer;
    animRecorded.splice(0, 9);

    stopButton.style.display = 'none';
    recButton.style.display = 'inline-block';
    playButton.style.display = 'inline-block';

    setTimeout(function() {
        for (const eng of engines){
             eng.speechState = [0.0,0.0,0.1,0.0,0.0];
        }
    }, 1000);
}
function play() {
    audioContext.resume();
//    console.log("play");
    playButton.style.display = 'none';
    recButton.style.display = 'none';
    const source = audioContext.createBufferSource();
//     console.log(source.context);
    source.buffer = recordedAudioBuffer;
    source.connect(audioContext.destination);
    source.start(0);
    processorNode.port.postMessage(true);
    let i = 0;
    processorNode.port.onmessage = (event) => {
        if (i < animRecorded.length){
            for (const eng of engines){
                 eng.speechState = animRecorded[i];
            }
        }else{
            playButton.style.display = 'inline-block';
            recButton.style.display = 'inline-block';
            processorNode.port.onmessage = null;
            processorNode.port.postMessage(false);
            setTimeout(function() {
                for (const eng of engines){
                     eng.speechState = [0.0,0.0,0.1,0.0,0.0];
                }
            }, 1000);
        }
        i+=1




       }

}

function concatenateFloat32Arrays(arrays) {
    // Calculate total length
    let totalLength = 0;
    for (let i = 0; i < arrays.length; i++) {
        totalLength += arrays[i].length;
    }

    // Create a new Float32Array with the total length
    let result = new Float32Array(totalLength);

    // Copy data from input arrays to the result array
    let offset = 0;
    for (let i = 0; i < arrays.length; i++) {
        result.set(arrays[i], offset);
        offset += arrays[i].length;
    }

    return result;
}
