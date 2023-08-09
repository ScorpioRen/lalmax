package rtc

import (
	config "lalmax/conf"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pion/ice/v2"
	"github.com/pion/webrtc/v3"
	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/naza/pkg/nazalog"
)

type RtcServer struct {
	config    config.RtcConfig
	lalServer logic.ILalServer
	udpMux    ice.UDPMux
	tcpMux    ice.TCPMux
}

func NewRtcServer(config config.RtcConfig, lal logic.ILalServer) (*RtcServer, error) {
	var udpMux ice.UDPMux
	var tcpMux ice.TCPMux

	if config.ICEUDPMuxPort != 0 {
		var udplistener *net.UDPConn
		udplistener, err := net.ListenUDP("udp", &net.UDPAddr{
			IP:   net.IP{0, 0, 0, 0},
			Port: config.ICEUDPMuxPort,
		})

		if err != nil {
			nazalog.Error(err)
			return nil, err
		}

		udpMux = webrtc.NewICEUDPMux(nil, udplistener)
	}

	if config.ICETCPMuxPort != 0 {
		var tcplistener *net.TCPListener
		tcplistener, err := net.ListenTCP("tcp", &net.TCPAddr{
			IP:   net.IP{0, 0, 0, 0},
			Port: config.ICETCPMuxPort,
		})

		if err != nil {
			nazalog.Error(err)
			return nil, err
		}

		tcpMux = webrtc.NewICETCPMux(nil, tcplistener, 20)
	}

	svr := &RtcServer{
		config:    config,
		lalServer: lal,
		udpMux:    udpMux,
		tcpMux:    tcpMux,
	}

	return svr, nil
}

func (s *RtcServer) HandleWHIP(c *gin.Context) {
	streamid := c.Request.URL.Query().Get("streamid")
	if streamid == "" {
		c.Status(http.StatusMethodNotAllowed)
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		nazalog.Error(err)
		c.Status(http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		nazalog.Error("invalid body")
		c.Status(http.StatusNoContent)
		return
	}

	pc, err := newPeerConnection(s.config.ICEHostNATToIPs, s.udpMux, s.tcpMux)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	whipsession := NewWhipSession(streamid, pc, s.lalServer)
	if whipsession == nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	sdp := whipsession.GetAnswerSDP(string(body))
	if sdp == "" {
		c.Status(http.StatusInternalServerError)
		return
	}

	go whipsession.Run()

	c.Data(http.StatusCreated, "application/sdp", []byte(sdp))
}

func (s *RtcServer) HandleWHEP(c *gin.Context) {
	streamid := c.Request.URL.Query().Get("streamid")
	if streamid == "" {
		c.Status(http.StatusMethodNotAllowed)
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		nazalog.Error(err)
		c.Status(http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		nazalog.Error("invalid body")
		c.Status(http.StatusNoContent)
		return
	}

	pc, err := newPeerConnection(s.config.ICEHostNATToIPs, s.udpMux, s.tcpMux)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	whepsession := NewWhepSession(streamid, pc, s.lalServer)
	if whepsession == nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	sdp := whepsession.GetAnswerSDP(string(body))
	if sdp == "" {
		c.Status(http.StatusInternalServerError)
		return
	}

	go whepsession.Run()

	c.Data(http.StatusCreated, "application/sdp", []byte(sdp))
}

func (s *RtcServer) handleWHEP(w http.ResponseWriter, r *http.Request, streamid, body string) {
	pc, err := newPeerConnection(s.config.ICEHostNATToIPs, s.udpMux, s.tcpMux)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	whepsession := NewWhepSession(streamid, pc, s.lalServer)
	if whepsession == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sdp := whepsession.GetAnswerSDP(string(body))
	if sdp == "" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	go whepsession.Run()

	w.Header().Set("Content-Type", "application/sdp")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(sdp))
}
