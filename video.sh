gst-launch-1.0 v4l2src device=/dev/video1 ! videorate ! "video/x-raw,framerate=3/1" ! 'jpegenc'  ! filesink buffer-size=1 location=/dev/stdout | ./video 

