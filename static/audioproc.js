
class SpeechNN{
    constructor() {
        this.wav2melModel = null;
        this.mel2phonModel = null;
        this.phon2avecModel = null;
        this.isReady = false;
        this.readyCallback = null;
        this.loadNetworks();
    }
    async loadNetworks(){
        this.wav2melModel = await tf.loadLayersModel('/file/wav2mel/model.json');
        this.mel2phonModel = await tf.loadLayersModel('/file/mel2phon/model.json');
        this.phon2avecModel = await tf.loadLayersModel('/file/phon2avec/model.json');

        const inputData = tf.tensor2d([new Float32Array(800)]);
        const melPredicted = this.wav2melModel.predict(inputData);
        const melTensors = [];
        for (let i = 0; i < 20; i++) {
            melTensors.push(melPredicted);
        }
        const melGroup = tf.concat(melTensors,1);
        const phoneGroup = this.mel2phonModel.predict(melGroup);
        const start = [0, 0, 0, 0];
        const end = [phoneGroup.shape[0], phoneGroup.shape[1],phoneGroup.shape[2]-1,phoneGroup.shape[3]];
        const phoneGroup1 = tf.slice(phoneGroup,start,end);
        const avec = this.phon2avecModel.predict(phoneGroup1);

        this.isReady = true;
        if (this.readyCallback) {
            this.readyCallback();
        }
    }

    async awaitResources() {
        if (this.isReady) {
          return Promise.resolve();
        }

        return new Promise((resolve) => {
          this.readyCallback = resolve;
        });
    }
}


class AnimCalc{
    constructor(speechNetworks) {
        this.speechNetworks = speechNetworks;
        this.melTensors = [];
        this.zeroAnimState = new Float32Array(5);
        this.zeroAnimState[0] = 0.5;
        this.zeroAnimState[1] = 0.48;
        this.zeroAnimState[2] = 0.52;
        this.zeroAnimState[3] = 0.43;
        this.zeroAnimState[4] = 0.46;
        var converterType = LibSampleRate.ConverterType.SRC_SINC_FASTEST;
    //        var converterType = LibSampleRate.ConverterType.SRC_SINC_BEST_QUALITY;
        this.resampler;
        let nChannels = 1;
        let inputSampleRate = audioContext.sampleRate;
        let outputSampleRate = 16000;

        LibSampleRate.create(nChannels, inputSampleRate, outputSampleRate, {
                converterType: converterType, // default SRC_SINC_FASTEST. see API for more
            }).then((src) => {

                this.resampler = src;
//
        });
    }
    getAnim(aBuffer){
        const result = this.makeMel(this.resampler.simple(aBuffer));
        if (result) {
            return result;
        }else{
            return [0.0,0.0,0.0,0.0,0.0];
        }
    }
    makeMel(aBuffer){
//        tf.engine().startScope()
        const inputData = tf.tensor2d([aBuffer]);
        const mel = this.speechNetworks.wav2melModel.predict(inputData);


        this.melTensors.push(mel)

        if (this.melTensors.length == 20){

            const melGroup = tf.concat(this.melTensors,1);

            const phoneGroup = this.speechNetworks.mel2phonModel.predict(melGroup);
            const start = [0, 0, 0, 0];
            const end = [phoneGroup.shape[0], phoneGroup.shape[1],phoneGroup.shape[2]-1,phoneGroup.shape[3]];
            const phoneGroup1 = tf.slice(phoneGroup,start,end);
            const avec = this.speechNetworks.phon2avecModel.predict(phoneGroup1);
            const avecSlice = tf.slice(avec,[0,10,0,0],[1,1,5,1]);
            const avecArrayTF = avecSlice.arraySync()


            const avecArray = new Float32Array(5);
            for (let i = 0; i < 5; i++) {
                avecArray[i] = avecArrayTF[0][0][i][0];
            }
//            this.avecToBshpCoef(avecArray,rEngine,onAnimate);

            //Dispose tensors
                this.melTensors[0].dispose();
                inputData.dispose();
                melGroup.dispose();
                phoneGroup.dispose();
                phoneGroup1.dispose();
                avec.dispose()
                avecSlice.dispose();
            //-----
            this.melTensors.shift()
            return this.avecToBshpCoef(avecArray)
        }


    }
    avecToBshpCoef(nnAvec){
        const result = new Float32Array(5);
        const amp = -7;
         for (let i = 0; i < 5; i++) {
                result[i] =amp*(nnAvec[i]-this.zeroAnimState[i]);
         }
         return result;
//         onAnimate(result,rEngine);

    }
}

/*

class AudioDrivenAnim {
    constructor(speechNetworks,audioContext,stream) {
        this.speechNetworks = speechNetworks;

        this.rEngine = null;
        this.bufferWritePosition = 0
        this.audioBuffer = new Float32Array(800);
        this.melTensors = [];
        this.zeroAnimState = new Float32Array(5);
        this.zeroAnimState[0] = 0.5;
        this.zeroAnimState[1] = 0.48;
        this.zeroAnimState[2] = 0.52;
        this.zeroAnimState[3] = 0.43;
        this.zeroAnimState[4] = 0.46;

        var converterType = LibSampleRate.ConverterType.SRC_SINC_FASTEST;
    //        var converterType = LibSampleRate.ConverterType.SRC_SINC_BEST_QUALITY;
        var resampler;
        let nChannels = 1;
        let inputSampleRate = audioContext.sampleRate;
        let outputSampleRate = 16000;

        LibSampleRate.create(nChannels, inputSampleRate, outputSampleRate, {
                converterType: converterType, // default SRC_SINC_FASTEST. see API for more
            }).then((src) => {

                resampler = src;
//
        });

        const audioSourceNode = stream;
        this.speechSyncNode = audioContext.createDelay();
        this.speechSyncNode.delayTime.value = 0.55;
        audioSourceNode.connect(this.speechSyncNode);


        audioContext.audioWorklet.addModule("audio_processor.js").then(
        () => {
            const processorNode = new AudioWorkletNode(
              audioContext,
              "my-audio-processor",
            );
//            var counter = 0;
            processorNode.port.onmessage = (event) => {


                const audioBuffer = event.data;
//                console.log(audioBuffer);
                if (resampler && this.rEngine.speechState){
                    const speechState = this.makeMel(resampler.simple(audioBuffer));
                    if (speechState){
                        this.rEngine.speechState = speechState;}
                }

//                    if ((counter%2) == 0){
//                        speechWorker.postMessage([2,audioBuffer]);
//                    }

//                    counter+=1;
              };
             audioSourceNode.connect(processorNode);
        }
        );

//    });


    }
    */
/*addBuffer(resampled){

        var elementsLeft = 800 - this.bufferWritePosition;
        if (elementsLeft > resampled.length) {
            elementsLeft = resampled.length;
            this.audioBuffer.set(resampled.subarray(0,elementsLeft), this.bufferWritePosition);
            this.bufferWritePosition += elementsLeft;

        }else if (elementsLeft == resampled.length){
            this.audioBuffer.set(resampled.subarray(0,elementsLeft), this.bufferWritePosition);
            this.bufferWritePosition = 0
            const bufferCopy = new Float32Array(800)
            bufferCopy.set(this.audioBuffer,0)

            this.makeMel(bufferCopy,rEngine,onAnimate)

        }else{
            this.audioBuffer.set(resampled.subarray(0,elementsLeft), this.bufferWritePosition);
            const elementToNewBuffer = resampled.length - elementsLeft;
            const bufferCopy = new Float32Array(800)
            bufferCopy.set(this.audioBuffer,0)

            this.makeMel(bufferCopy,rEngine,onAnimate)


            this.audioBuffer.set(resampled.subarray(elementsLeft,resampled.length), 0);
            this.bufferWritePosition = resampled.length - elementsLeft;

         }


    }*//*

    makeMel(aBuffer){
//        tf.engine().startScope()
        const inputData = tf.tensor2d([aBuffer]);
        const mel = this.speechNetworks.wav2melModel.predict(inputData);


        this.melTensors.push(mel)

        if (this.melTensors.length == 20){

            const melGroup = tf.concat(this.melTensors,1)

            const phoneGroup = this.speechNetworks.mel2phonModel.predict(melGroup);
            const start = [0, 0, 0, 0];
            const end = [phoneGroup.shape[0], phoneGroup.shape[1],phoneGroup.shape[2]-1,phoneGroup.shape[3]];
            const phoneGroup1 = tf.slice(phoneGroup,start,end);
            const avec = this.speechNetworks.phon2avecModel.predict(phoneGroup1);
            const avecSlice = tf.slice(avec,[0,10,0,0],[1,1,5,1]);
            const avecArrayTF = avecSlice.arraySync()


            const avecArray = new Float32Array(5);
            for (let i = 0; i < 5; i++) {
                avecArray[i] = avecArrayTF[0][0][i][0];
            }
//            this.avecToBshpCoef(avecArray,rEngine,onAnimate);

            //Dispose tensors
                this.melTensors[0].dispose();
                inputData.dispose();
                melGroup.dispose();
                phoneGroup.dispose();
                phoneGroup1.dispose();
                avec.dispose()
                avecSlice.dispose();
            //-----
            this.melTensors.shift()
            return this.avecToBshpCoef(avecArray)
        }


    }
    avecToBshpCoef(nnAvec){
        const result = new Float32Array(5);
        const amp = -7;
         for (let i = 0; i < 5; i++) {
                result[i] =amp*(nnAvec[i]-this.zeroAnimState[i]);
         }
         return result;
//         onAnimate(result,rEngine);

    }

}*/
