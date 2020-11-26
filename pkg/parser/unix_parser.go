package parser

import (
	"fmt"
	"io/ioutil"

	"github.com/crowdsecurity/crowdsec/pkg/csconfig"

	"github.com/logrusorgru/grokky"
	log "github.com/sirupsen/logrus"
)

type UnixParserCtx struct {
	Grok       grokky.Host
	Stages     []string
	Profiling  bool
	DataFolder string
}

type Parsers struct {
	Ctx             *UnixParserCtx
	Povfwctx        *UnixParserCtx
	StageFiles      []Stagefile
	PovfwStageFiles []Stagefile
	Nodes           []Node
	Povfwnodes      []Node
	EnricherCtx     []EnricherCtx
}

func Init(c map[string]interface{}) (*UnixParserCtx, error) {
	r := UnixParserCtx{}
	r.Grok = grokky.NewBase()
	files, err := ioutil.ReadDir(c["patterns"].(string))
	if err != nil {
		return nil, err
	}
	r.DataFolder = c["data"].(string)
	for _, f := range files {
		if err := r.Grok.AddFromFile(c["patterns"].(string) + f.Name()); err != nil {
			log.Errorf("failed to load pattern %s : %v", f.Name(), err)
			return nil, err
		}
	}
	log.Debugf("Loaded %d pattern files", len(files))
	return &r, nil
}

func LoadParsers(cConfig *csconfig.GlobalConfig, parsers *Parsers) (*Parsers, error) {
	var err error

	log.Infof("Loading grok library %s", cConfig.Crowdsec.ConfigDir+string("patterns/"))
	/* load base regexps for two grok parsers */
	parsers.Ctx, err = Init(map[string]interface{}{"patterns": cConfig.Crowdsec.ConfigDir + string("patterns/"),
		"data": cConfig.Crowdsec.DataDir})
	if err != nil {
		return parsers, fmt.Errorf("failed to load parser patterns : %v", err)
	}
	parsers.Povfwctx, err = Init(map[string]interface{}{"patterns": cConfig.Crowdsec.ConfigDir + string("patterns/"),
		"data": cConfig.Crowdsec.DataDir})
	if err != nil {
		return parsers, fmt.Errorf("failed to load postovflw parser patterns : %v", err)
	}

	/*
		Load enrichers
	*/
	log.Infof("Loading enrich plugins")

	parsers.EnricherCtx, err = Loadplugin(cConfig.Crowdsec.DataDir)
	if err != nil {
		return parsers, fmt.Errorf("Failed to load enrich plugin : %v", err)
	}

	/*
	 Load the actual parsers
	*/

	log.Infof("Loading %d parsers", len(parsers.StageFiles))

	parsers.Nodes, err = LoadStages(parsers.StageFiles, parsers.Ctx, parsers.EnricherCtx)
	if err != nil {
		return parsers, fmt.Errorf("failed to load parser config : %v", err)
	}

	if len(parsers.PovfwStageFiles) > 0 {
		log.Infof("loading %d postoverflow parsers", len(parsers.PovfwStageFiles))
	} else {
		log.Infof("no postoverflow parsers to load")
	}
	parsers.Povfwnodes, err = LoadStages(parsers.PovfwStageFiles, parsers.Povfwctx, parsers.EnricherCtx)

	if err != nil {
		return parsers, fmt.Errorf("failed to load postoverflow config : %v", err)
	}

	if cConfig.Prometheus != nil && cConfig.Prometheus.Enabled {
		parsers.Ctx.Profiling = true
		parsers.Povfwctx.Profiling = true
	}

	return parsers, nil
}
