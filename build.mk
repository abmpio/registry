
LOCAL_PATH:= $(call my-dir)

include $(CLOUDMOBILE_CLEAR_VARS)

LOCAL_MODULE := registry.consul
LOCAL_SOURCE := plugin/plugin.go
LOCAL_TARGET := registry.consul.so

include $(CLOUDMOBILE_BUILD_GO_LIBRARY)
