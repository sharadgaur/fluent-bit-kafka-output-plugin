package main

import (
  "github.com/fluent/fluent-bit-go/output"
  "github.com/ugorji/go/codec"
  "github.com/Shopify/sarama"
  "encoding/json"
  "reflect"
  "unsafe"
  "fmt"
  "io" 
  "C"
)

//export FLBPluginInit
func FLBPluginInit(ctx unsafe.Pointer) int {
    return output.FLBPluginRegister(ctx, "out_kafka", "out_kafka GO!")
}

//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
  var h codec.Handle = new(codec.MsgpackHandle)
  var b []byte
  var m interface{}
  var err error
  var enc_data []byte

  b = C.GoBytes(data, length)
  dec := codec.NewDecoderBytes(b, h)

  // Iterate the original MessagePack array
  for {
    // Decode the msgpack data
    err = dec.Decode(&m)
    if err != nil {
      if err == io.EOF {
        break
      }
      fmt.Printf("Failed to decode msgpack data: %v\n", err)
      return output.FLB_ERROR
    }

    // select format until config files are available for fluentbit
    format := "json"

    if format == "json" {
      enc_data, err = encode_as_json(m)
      //enc_data, err = m
    } else if format == "msgpack" {
      enc_data, err = encode_as_msgpack(m)
    } else if format == "string" {
      // enc_data, err == encode_as_string(m)
    }
    if err != nil {
      fmt.Printf("Failed to encode %s data: %v\n", format, err)
      return output.FLB_ERROR
    }
    brokerList := []string{"kafka-0.kafka.default.svc.cluster.local:9092"}
    producer, err := sarama.NewSyncProducer(brokerList, nil)

    if err != nil {
      fmt.Printf("Failed to start Sarama producer: %v\n", err)
      return output.FLB_ERROR
    }

    producer.SendMessage(&sarama.ProducerMessage {
      Topic: "logs_default",
      Key:   nil,
      Value: sarama.ByteEncoder(enc_data),
    })

    producer.Close()
  }
  return output.FLB_OK
}

func encode_as_json(m interface {}) ([]byte, error) {
  slice := reflect.ValueOf(m)
  timestamp := slice.Index(0).Interface().(uint64)
  record := slice.Index(1).Interface().(map[interface{}] interface{})

  record2 := make(map[string] interface{})
  for k, v := range record {
    if val, ok := v.([]byte); ok {
      v2 := string(val)
      record2[k.(string)] = v2
    } else {
      record2[k.(string)] = v
    }
  }

  type Log struct {
    Time uint64
    Record map[string] interface{}
  }

  log := Log {
    Time: timestamp,
    Record: record2,
  }

  return json.Marshal(log)
}


func encode_as_msgpack(m interface {}) ([]byte, error) {
  var (
    mh codec.MsgpackHandle
    w io.Writer
    b []byte
  )

  enc := codec.NewEncoder(w, &mh)
  enc = codec.NewEncoderBytes(&b, &mh)
  err := enc.Encode(&m)
  return b, err
}

func FLBPluginExit() int {
  return 0
}

func main() {
}
